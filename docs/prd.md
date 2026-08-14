# Product Requirements Document (PRD)

## Mini Leave Management System

### 1. Product Overview

**Product Name:** Mini Leave Management System
**Product Type:** Internal web application
**Purpose:** Provide a simple system for employees to apply for leave and managers to approve or reject leave requests.

The project is intentionally small and will be used as a **mock project to demonstrate ICM (Interpretable Context Methodology)** and vertical-slice development.

---

## 2. Problem Statement

In a small organization, leave requests may be handled through messages, emails, or spreadsheets. This can cause:

* Lack of visibility into leave status
* Incorrect leave balances
* Difficulty tracking approvals
* Duplicate or overlapping leave requests
* Manual calculation of remaining leave

The system should provide a simple centralized workflow for managing employee leave.

---

## 3. Product Goal

Build a lightweight leave management application where:

> **An employee can apply for leave, a manager can approve/reject it, and the system automatically maintains the employee's leave balance.**

---

## 4. Target Users

### Employee

Can:

* Log in
* View leave balance
* Apply for leave
* View submitted requests
* View approval status
* View leave history

### Manager

Can:

* Log in
* View pending leave requests
* Review employee leave details
* Approve leave
* Reject leave
* View leave history

### Admin

For the mock project, Admin functionality will be minimal.

Admin can:

* Create employees
* Assign employee roles
* Create leave types
* Assign leave balances

---

# 5. Core Features

## 5.1 Authentication

Users should be able to log into the system.

**Requirements:**

* User enters email/username and password.
* System validates credentials.
* User is redirected according to their role.
* Unauthorized users cannot access protected pages.
* User can log out.

---

## 5.2 Employee Management

Admin should be able to manage employees.

**Requirements:**

* Create employee
* View employee
* Edit employee
* Assign role
* Activate/deactivate employee

Basic employee information:

```text
Employee
├── ID
├── Name
├── Email
├── Department
├── Role
└── Status
```

---

## 5.3 Leave Type Management

Admin should be able to configure leave types.

Example:

```text
Casual Leave
Sick Leave
Earned Leave
```

Each leave type should contain:

```text
Leave Type
├── ID
├── Name
├── Description
└── Annual Allocation
```

---

## 5.4 Leave Balance

Each employee should have a balance for each applicable leave type.

Example:

```text
Employee: Rahul

Casual Leave     8 / 12
Sick Leave       5 / 10
Earned Leave     12 / 15
```

The system should track:

* Allocated days
* Used days
* Remaining days

---

# 6. Apply Leave

This is the primary employee workflow.

Employee selects:

```text
Leave Type
From Date
To Date
Reason
```

The system calculates the number of requested leave days.

### Business Rules

1. From date cannot be after To date.
2. Leave type must be selected.
3. Reason is required.
4. Requested days must be greater than zero.
5. Employee must have sufficient available balance.
6. Employee cannot create conflicting leave requests.
7. New requests start with `Pending` status.

---

# 7. Leave Approval

Managers should be able to review pending requests.

Example:

```text
Pending Requests

Rahul
Casual Leave
10 Aug → 12 Aug
3 Days
Reason: Personal work

[Approve] [Reject]
```

### Approval Rules

* Only authorized managers can approve/reject requests.
* A pending request can be approved or rejected.
* An approved request cannot be approved again.
* A rejected request cannot be rejected again.
* Approval must update the leave balance.
* Rejection must not deduct leave balance.

---

# 8. Leave Status

Every leave request should have a status.

```text
Pending
Approved
Rejected
Cancelled
```

For the initial version, the primary workflow is:

```text
Pending
   │
   ├── Approve → Approved
   │
   └── Reject  → Rejected
```

Cancellation can remain optional for the first version.

---

# 9. Leave History

Employees should be able to view their previous requests.

Example:

| Leave Type | Dates     | Days | Status   |
| ---------- | --------- | ---: | -------- |
| Casual     | 10–12 Aug |    3 | Approved |
| Sick       | 15 Aug    |    1 | Rejected |
| Earned     | 20–21 Aug |    2 | Pending  |

Managers should also be able to view leave history for employees they manage.

---

# 10. Dashboard

### Employee Dashboard

Display:

```text
Welcome, Rahul

Leave Balance
-------------------------
Casual       8 days
Sick         5 days
Earned      12 days

Recent Requests
-------------------------
CL   10 Aug   Approved
SL   15 Aug   Pending
```

### Manager Dashboard

Display:

```text
Manager Dashboard

Pending Requests: 5
Approved Today:   3
Rejected Today:   1

Pending Leave Requests
-------------------------
Rahul       3 days
Priya       2 days
Amit        1 day
```

---

# 11. Roles & Permissions

| Feature              | Employee | Manager |    Admin |
| -------------------- | -------: | ------: | -------: |
| Login                |        ✅ |       ✅ |        ✅ |
| View Own Profile     |        ✅ |       ✅ |        ✅ |
| Apply Leave          |        ✅ |       ✅ | Optional |
| View Own Leave       |        ✅ |       ✅ |        ✅ |
| Approve Leave        |        ❌ |       ✅ | Optional |
| Reject Leave         |        ❌ |       ✅ | Optional |
| Manage Employees     |        ❌ |       ❌ |        ✅ |
| Manage Leave Types   |        ❌ |       ❌ |        ✅ |
| Manage Leave Balance |        ❌ |       ❌ |        ✅ |

---

# 12. Core Data Model

The initial database should contain:

```text
User
 ├── id
 ├── employee_id
 ├── username
 ├── password_hash
 └── role

Employee
 ├── id
 ├── name
 ├── email
 ├── department
 └── status

LeaveType
 ├── id
 ├── name
 ├── description
 └── annual_allocation

LeaveBalance
 ├── id
 ├── employee_id
 ├── leave_type_id
 ├── allocated_days
 └── used_days

LeaveRequest
 ├── id
 ├── employee_id
 ├── leave_type_id
 ├── from_date
 ├── to_date
 ├── days
 ├── reason
 ├── status
 ├── reviewed_by
 └── reviewed_at
```

---

# 13. Main Workflow

```text
Employee Login
      ↓
View Leave Balance
      ↓
Apply Leave
      ↓
System Validates Request
      ↓
Create Pending Request
      ↓
Manager Login
      ↓
View Pending Requests
      ↓
Review Request
      ↓
 ┌───────────────┐
 │               │
Approve        Reject
 │               │
 ↓               ↓
Deduct        No Deduction
Balance          │
 │               │
 └───────┬───────┘
         ↓
    Update Status
         ↓
   Employee Sees Result
```

---

# 14. Non-Functional Requirements

### Performance

* Normal page requests should respond quickly.
* The system should support a small organization's data volume.

### Security

* Passwords must never be stored as plain text.
* Protected pages require authentication.
* Users must only access functionality allowed by their role.
* Server-side validation is required.

### Usability

* Simple navigation.
* Clear leave status.
* Clear validation messages.
* Mobile-friendly UI.

### Reliability

* Leave balance changes should happen atomically with approval.
* Failed operations should not leave inconsistent leave balances.

---

# 15. Technology Context

For this mock project, I recommend:

```text
Backend
   Go

Frontend
   HTMX
   HTML
   CSS

Database
   SQLite

Architecture
   Vertical Slice Architecture

Methodology
   ICM
```

You don't need React for this project. The purpose is to demonstrate **requirements → context → business rules → vertical slices → implementation**.

---

# 16. Vertical Slice Scope

The PRD maps to the 7 slices we defined:

### VS-01 — Project & Authentication

Deliver:

* Project setup
* Database connection
* User authentication
* Session handling
* Role-based access

### VS-02 — Employee Management

Deliver:

* Employee CRUD
* Employee roles
* Employee status

### VS-03 — Leave Types & Balance

Deliver:

* Leave type management
* Employee leave allocation
* Leave balance calculation

### VS-04 — Apply Leave

Deliver:

* Leave application form
* Date calculation
* Validation
* Balance validation
* Pending request creation

### VS-05 — Manager Approval

Deliver:

* Pending request list
* Request details
* Approve
* Reject
* Balance deduction

### VS-06 — Leave History & Dashboard

Deliver:

* Employee dashboard
* Manager dashboard
* Leave history
* Status display

### VS-07 — Rules & Integration

Deliver:

* Cross-feature validation
* Overlapping leave detection
* Transaction handling
* End-to-end workflow
* Integration tests
* Final cleanup

---

# 17. ICM Context Structure

This is the important part for your experiment.

The PRD should become the foundation for your ICM artifacts:

```text
PRD
 │
 ├── Business Context
 │
 ├── User Context
 │
 ├── Business Goals
 │
 ├── Actors
 │
 ├── Features
 │
 ├── Business Rules
 │
 ├── Workflows
 │
 ├── Data Context
 │
 └── Technical Context
        │
        ▼
   Vertical Slice Index
        │
        ▼
   Individual Slice Context
        │
        ▼
   Development
```

So **don't start coding directly from this PRD**.

The next logical ICM step is to convert this PRD into:

**Business Goals → Business Rules → Atomic Requirements → BPMN/Workflow → Vertical Slice Index → Detailed Vertical Slice Specifications.**

That will give you a complete small project where you can actually demonstrate how ICM works from **idea to working software**.

**Confidence: 96%**
**Reason:** The PRD is intentionally scoped to the 7-slice mock system we defined and keeps the requirements small enough for an ICM demonstration while still containing meaningful business rules and workflow.
