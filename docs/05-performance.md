# Measured latency

Numbers, the conditions they were taken under, and the commands that reproduce
them. The README quotes the headline figures and points here.

## Reproducing

```bash
docker compose up -d
go run ./cmd/seed -n 200000      # from api/, about 26 seconds
go run ./cmd/bench               # from api/, 50 requests per query
```

The seed is fixed at 42, so the same data comes back every run. `-n` changes the
size; `-runs` changes how many requests each query gets.

## Conditions

Everything below was measured on one machine, so treat the numbers as the shape
of the thing rather than as a service level.

| | |
|---|---|
| Host | AMD Ryzen 7 6800HS, 16 logical cores, 16GB |
| Database | PostgreSQL 17.10 in Docker, default configuration |
| Client | The API and the benchmark on the same host, one request at a time |
| Cache | Warm. One unmeasured request per query first |

One client and a warm cache is the friendly case. It is the right one for asking
whether the query plans hold at size, which is what the indexes were built for,
and the wrong one for asking what the service does under concurrent load. That
second question is not answered here.

## Results

Measured over 50 requests per query, through HTTP, reading the whole response
body. Page size is the default 50.

### 200,000 tasks

14,115 blocked, 800 in the trash.

| Query | p50 | p95 | max |
|---|---|---|---|
| Default list | 1.0ms | 1.6ms | 2.2ms |
| Sorted by due | 1.0ms | 1.5ms | 1.9ms |
| Sorted by name | 1.0ms | 1.3ms | 1.4ms |
| Sorted by priority | 1.0ms | 1.5ms | 1.6ms |
| Sorted by status | 1.0ms | 1.8ms | 2.6ms |
| Filtered by status | 1.0ms | 1.8ms | 2.7ms |
| Overdue | 1.1ms | 2.0ms | 3.1ms |
| Blocked | 1.0ms | 2.6ms | 2.8ms |
| Blocked, sorted by due | 1.6ms | 3.0ms | 3.0ms |
| Name search | 4.4ms | 6.0ms | 7.5ms |
| Counts | 31.1ms | 42.7ms | 43.8ms |

### 20,000 tasks

| Query | p50 | p95 | max |
|---|---|---|---|
| Default list | 1.0ms | 2.2ms | 2.5ms |
| Sorted by due | 0.9ms | 1.4ms | 1.6ms |
| Blocked, sorted by due | 1.2ms | 2.1ms | 2.2ms |
| Name search | 2.5ms | 4.1ms | 4.5ms |
| Counts | 5.0ms | 6.4ms | 9.4ms |

## What the numbers say

**The list does not notice the tenfold increase.** Every paged query is within a
millisecond of itself at 20,000 and 200,000 rows, because a keyset page reads
fifty-one index entries whatever the table holds behind them. An offset would
have made page four hundred cost four hundred times page one; this is what that
decision bought.

**Name search is the one filter that scans**, and it shows: four times the cost
of a seek, and the only paged query that grew with the table. A leading wildcard
cannot use a btree index, so it is a trigram bitmap scan followed by a sort. The
client holds the query until three characters, which keeps the least selective
searches off the database entirely.

**The counts query is the outlier, and it has to be.** A count must see every
matching row, so the partial indexes that make the list cheap do not help. Its
median went from 5.0ms to 31.1ms with the data, roughly linearly, which is the
expected shape for a scan. That is the argument for asking once and holding the answer
for thirty seconds on the client rather than recounting after every keystroke.

**Blocked with a sort is not the worst case the plan predicted.** The plan named
it as the query most likely to degrade. At 200,000 rows it is 3.0ms, because the
planner serves it from the sort key's index and filters, discarding a few
hundred rows to fill a page. It would get worse the rarer blocked became; at 7%
of the table it does not.

## Where this stops being true

Three limits worth naming rather than leaving for someone to find.

- **Concurrency is not measured.** One client at a time says nothing about
  connection pool contention or lock waits.
- **The blocked partial indexes follow the default sort only.** Combined with
  another sort the planner falls back to filtering, which is cheap here and
  would not be if blocked were rare.
- **Counts grows with the table.** At ten million rows a scan that already takes
  31ms at the median becomes a problem, and the answer is a maintained summary
  rather than a longer cache.
