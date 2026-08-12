# The brief: what SleekFlow actually asked for

Source: `SleekFlow Software Engineer Interview Project.pdf`

This file is the requirements only. No interpretation, no decisions. Those live in
[`DECISIONS.md`](../DECISIONS.md) and [`01-requirements-interpretation.md`](./01-requirements-interpretation.md).
If the two ever disagree, this file is what the client asked for and those are what I chose to do
about it.

---

## Objective

Build a TODO list web application with a **backend API** and a **simple web UI**.

The brief states its own evaluation focus: interpreting requirements, making architectural
decisions, prioritising, and delivering a working solution with clear reasoning.

---

## Core features (required)

### A task holds

| Field | Values |
|---|---|
| Unique ID | None given |
| Name | None given |
| Description | None given |
| Due date | None given |
| Status | Not Started · In Progress · Completed · Archived |
| Priority | Low · Medium · High ("e.g.", so the set is a suggestion) |

Standard CRUD: create, read, update, delete.

### Recurring tasks

- Tasks can recur: **daily, weekly, monthly, or custom**.
- When a recurring task is marked completed, **the next occurrence is created automatically**
  based on its schedule.

### Task dependencies

- A task can depend on **one or more** other tasks.
- A dependent task **cannot be moved to "In Progress"** until all of its dependencies are
  "Completed".

### Filtering and sorting

- **Filter by:** status, priority, due date, dependency status (blocked / unblocked)
- **Sort by:** due date, priority, status, name

### Web UI

A simple, functional interface for creating, editing, deleting, filtering and sorting.
The brief explicitly says: *"The UI does not need to be visually polished; functional and
usable is sufficient."*

---

## Non-functional requirements

1. The API should support **multiple users accessing the same TODO list concurrently**.
2. **Data should not be permanently lost** when a TODO is deleted.
3. The system should handle **10,000+ items without degrading user experience**.

---

## Nice-to-have (optional)

The brief says: *"These are optional. Prioritize core features first."*

- User authentication and registration
- Real-time updates across browser tabs or users
- Bulk operations (e.g. complete all tasks in a group)
- DevOps setup (e.g. Docker, CI/CD)
- Architecture diagram
- Any other improvements or features you see fit

---

## Deliverables

1. A **GitHub repository** with the source code.
2. A **README** with setup and local development instructions.
3. **API documentation**. Swagger/OpenAPI, Postman collection, or equivalent.
4. A **decision log (1–2 pages)** in the repository covering:
   - how ambiguous or underspecified requirements were interpreted, and the reasoning
   - key architectural decisions and the trade-offs considered
   - what was chosen **not** to be built, and why
   - what would be done differently with more time
5. A **live demo** during the interview, followed by discussion. Environment ready and the
   app running locally beforehand.

---

## Guidelines

- Any modern backend language/framework (ASP.NET Core, Flask/Django, Express, Spring Boot, Go, …)
- Any frontend framework (React, Angular, Vue, …)
- Any suitable database (PostgreSQL, MongoDB, MySQL, MSSQL, …)
- Implement **error handling and input validation** for API requests
- Write **tests for core functionality** (unit and/or integration)
- Ensure the application **can be easily run and tested locally**
- AI coding tools are welcome. **AI session transcripts may optionally be included** as
  supplementary material. Not required, but welcomed.

---

## A note on scope (verbatim, because it changes everything)

> "This project is designed to have more requirements than you may be able to fully implement
> in a reasonable timeframe. **This is intentional.** We are not expecting you to build
> everything. How you prioritize, what you choose to focus on, and how you communicate those
> decisions in your decision log are all part of the evaluation."

---

## What they say they're evaluating

| Criterion | Their words |
|---|---|
| Requirement interpretation | Did you identify areas where requirements are ambiguous or underspecified? How did you resolve them? |
| Planning and prioritization | Did you scope the work sensibly? Did you focus on core features before nice-to-haves? |
| Technical decisions | Are your architectural choices well-reasoned? Can you articulate the trade-offs? |
| Code quality | Is the code well-structured, testable, and maintainable? |
| Verification | Do your tests cover meaningful scenarios, including edge cases? |
| Communication | Is your decision log clear and well-reasoned? Can you walk us through your choices in the live demo? |
