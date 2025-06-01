# Reconciliation System

A **highly scalable system** designed to automate the **matching and validation of transactions** across multiple CSV data sources (internal vs. external).

---

## Table of Contents

* [1. Project Overview](#1-project-overview)
* [2. Assumptions](#2-assumptions)
* [3. System Architecture](#3-system-architecture)
* [4. Database Models](#4-database-models)
* [5. Key Features](#5-key-features)
* [6. Prerequisites](#6-prerequisites)
* [7. How to Run & Test](#7-how-to-run--test)
* [8. Makefile Commands](#8-makefile-commands)
* [9. Limitations & Future Improvements](#9-limitations--future-improvements)

---

## 1. Project Overview

This system implements a **Transaction Reconciliation Platform** to automate the comparison between:

* Internal **transaction records**
* External **bank statement entries**

The platform is composed of:

*  **HTTP Server** — REST API for triggering and querying reconciliations
*  **Cron Worker** — Periodic background jobs for continuous reconciliation

> **Database:** PostgreSQL (shared by both components)

---

## 2. Assumptions

* All data comes in **CSV** format.
* **Timestamps** are in **RFC3339 (UTC / Z)** format.
* **Amounts** can be negative (to represent debits).
* Inputs are valid local paths to the CSV files.
* Configurable runtime via **environment variables**.
* Matching logic uses **transaction ID**, **amount**, and **timestamp range**.

---

## 3. System Architecture

### System Flow Diagram
![reconcile_app drawio (1)](https://github.com/user-attachments/assets/dc4c5305-8530-4a26-9d50-710e5c288335)

#### Process Reconciliation (HTTP)
1. Receive and validate the reconciliation request payload.
2. Upload the CSV files (transaction and reference data).
3. If validation and upload are successful, create a new reconciliation process log and associated asset records in the database.
  
#### Get Result (HTTP)
1. Get the reconciliation result using the provided log_id.
2. Retrieve the corresponding process log from the database and return the result, including status and summary if available.
  
#### Reconcile execution (CRON)
1. Periodically checks for pending reconciliation processes with status INIT or RUNNING.
2. Uses an in-memory cache or distributed lock to ensure the process isn't already being executed by another worker.
3. If the process is not locked, it executes the reconciliation logic using the associated CSV file links.
4. After processing, updates the process log with the results and sets the final status (e.g., RUNNING, FINISH).


### HTTP Server

| Endpoint                       | Description                             |
| ------------------------------ | --------------------------------------- |
| `POST /process_reconciliation` | Trigger a reconciliation with CSV input |
| `GET /get_result?log_id={id}`  | Get reconciliation results by log ID    |

* Validates and parses input
* Converts dates to UNIX timestamps
* Delegates to business logic (usecase layer)

### Cron Worker

* Runs at configurable intervals
* Spawns **N workers** (parallel processing)
* Uses a **distributed lock** to prevent concurrent runs

### PostgreSQL Database

* Stores process metadata and results
* Migrates automatically on startup

---

## 4. Database Models

### ReconciliationProcessLog

Stores metadata about each reconciliation job.

| Field              | Type   | Description                            |
| ------------------ | ------ | -------------------------------------- |
| ID                 | int64  | Auto-increment primary key             |
| ReconciliationType | int64  | 1 = Bank Transaction                   |
| TotalMainRow       | int64  | Expected transactions to be processed  |
| CurrentMainRow     | int64  | Actual transactions processed so far   |
| ProcessInfo        | string | JSON-encoded metadata                  |
| Status             | int    | 1 = Init, 2 = Running, 3 = Success     |
| Result             | string | JSON summary of results                |
| CreateTime         | int64  | UNIX timestamp                         |
| CreateBy           | string | Operator                               |
| UpdateTime         | int64  | Last update timestamp                  |
| UpdateBy           | string | Operator                               |

### ReconciliationProcessLogAsset

Each reconciliation job can have multiple assets (CSV files).

| Field                      | Type   | Description                         |
| -------------------------- | ------ | ----------------------------------- |
| ID                         | int64  | Auto-increment primary key          |
| ReconciliationProcessLogID | int64  | Foreign key to the main log         |
| DataType                   | int64  | 1 = Transaction, 2 = Bank Statement |
| FileName                   | string | Original name of uploaded file      |
| FileUrl                    | string | File path or URL                    |
| CreateTime                 | int64  | UNIX timestamp                      |
| CreateBy                   | string | Uploader identity                   |

#### `ProcessInfo` JSON Format

```json
{
  "start_time": 1717200000,
  "end_time": 1717286399
}
```

#### `Result` JSON Format

```json
{
  "total_processed": 3,
  "matched": 2,
  "unmatched": 1,
  "system_unmatched": [...],
  "bank_unmatched_by_source": {
    "bank_statement.csv": [...]
  },
  "total_discrepancy": 20000
}
```

---

## 5. Key Features

* Bulk reconciliation with progress tracking
* Configurable concurrency and interval
* Supports resume on failure: Automatically resumes processing from the last completed step if the system goes down mid-process
* Flexible time window for reconciliation
* Transparent logging and result storage
* Docker-based local development setup
* Clean project separation: handler, usecase, DAO, model

---

## 6. Prerequisites

Ensure the following are installed:

* **Docker** (v20.10+)
* **Docker Compose** (v1.29+)
* **Go** (v1.14+) *(optional: for manual builds)*
* **PostgreSQL** (or run via Docker)

> Use a `.env` file to store environment variables.

---

## 7. How to Run & Test

### 1. Start the stack

```bash
make run
```

### 2. Trigger reconciliation (sample)

```bash
curl -X POST http://localhost:8080/process_reconciliation \
  -H "Content-Type: application/json" \
  -d '{
        "transaction_csv_path": "data/transactions.csv",
        "reference_csv_paths": ["data/bank_statement.csv"],
        "start_date": "2024-06-01",
        "end_date": "2024-06-01",
        "operator": "radhian"
      }'
```

### 3. Get reconciliation result

```bash
curl "http://localhost:8080/get_result?log_id=810"
```

### Result (formatted)

```json
{
  "total_processed": 3,
  "matched": 2,
  "unmatched": 1,
  "system_unmatched": [
    {
      "TrxID": "TRX003",
      "Amount": 15000,
      "Type": "CREDIT",
      "TransactionTime": "2024-06-01T08:30:00Z"
    }
  ],
  "bank_unmatched_by_source": {
    "bank_statement.csv": [
      {
        "UniqueIdentifier": "BRX004",
        "Amount": 5000,
        "Date": "2024-06-01T00:00:00Z"
      }
    ]
  },
  "total_discrepancy": 20000
}
```

### 4. Try large CSVs

```bash
curl -X POST http://localhost:8080/process_reconciliation \
  -H "Content-Type: application/json" \
  -d '{
        "transaction_csv_path": "data/transactions_large.csv",
        "reference_csv_paths": ["data/bank_statement_large.csv"],
        "start_date": "2024-06-01",
        "end_date": "2024-06-01",
        "operator": "radhian"
      }'
```

---

## 8. Makefile Commands

Use the following commands to manage your environment:

| Command        | Description                    |
| -------------- | ------------------------------ |
| `make run`     | Build and start all services   |
| `make down`    | Stop and clean up all services |
| `make restart` | Rebuild and restart services   |

### Example Usage

```bash
make run       # Start the application
make down      # Tear everything down
make restart   # Full rebuild and restart
```

## 9. Limitations & Future Improvements
#### 1. Local File Storage for CSV Uploads
CSV files are currently uploaded and stored in the local file system.
*  ⚠️ This limits scalability and portability, especially in containerized or distributed deployments.
Future Improvement: Integrate with cloud object storage services such as AWS S3, Google Cloud Storage, etc to support remote, scalable, and durable file uploads.

#### 2. Static Configuration via .env File
Application settings (e.g., database config, batch size, worker count) are loaded from a .env file at startup.

* ⚠️ Any configuration change requires a full application restart to take effect.
Future Improvement: Implement dynamic or hot-reloadable configuration using tools like Viper, Consul, or environment watchers.
