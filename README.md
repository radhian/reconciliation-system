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

### General
* All data comes in **CSV** format.
* **Timestamps** are in **RFC3339 (UTC / Z)** format.
* **Amounts** can be negative (to represent debits).
* Inputs are valid local paths to the CSV files.
* Configurable runtime via **environment variables**.

### Matching Logic

The reconciliation service matches transactions between the internal system and bank statements using a combination of **transaction type**, **amount**, and **count of transactions within each grouped key(TransactionType-Amount)**.

#### Key Concepts:

- **Transaction Type Mapping:**
  - Internal transactions use `"CREDIT"` or `"DEBIT"` mapped to single-letter codes:  
    - `"CREDIT"` → `"c"`  
    - `"DEBIT"` → `"d"`  
    - Unknown types default to `"u"`
  - Bank statement amounts are signed:
    - Negative amounts → `"d"` (debit)  
    - Positive amounts → `"c"` (credit)

- **Grouping by Key:**  
  Each transaction or bank entry is grouped by a **key** formatted as:
{typeCode}|{absolute_amount}

For example, a credit transaction of 100.00 is grouped as `"c|100.00"`.

- **Count-Based Matching:**  
Within each group key, transactions from both sides are matched based on the minimum count between internal transactions and bank statements.
- If the internal system has 5 transactions in group `"c|100.00"` and the bank has 3 entries in the same group, only 3 are matched.
- The leftover 2 internal transactions are considered unmatched.
- Likewise, any leftover bank entries after matching are also marked unmatched.

#### Example

| Transaction ID | Amount | Type   | Group Key  |
|----------------|--------|--------|------------|
| T001           | 100.00 | CREDIT | c|100.00   |
| T002           | 100.00 | CREDIT | c|100.00   |
| T003           | 200.50 | DEBIT  | d|200.50   |

| Bank Statement ID | Amount  | Group Key  |
|-------------------|---------|------------|
| B001              | 100.00  | c|100.00   |
| B002              | -200.50 | d|200.50   |
| B003              | 100.00  | c|100.00   |
| B004              | -300.00 | d|300.00   |

- **Matched:**  
- T001 ↔ B001 (c|100.00)  
- T002 ↔ B003 (c|100.00)  
- T003 ↔ B002 (d|200.50)

- **Unmatched:**  
- Bank: B004 (d|300.00)  
- System: None


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
* **PostgreSQL** (run via Docker)

> Please check a `.env` file to store environment variables.

---

## 7. How to Run & Test

### 1. Start the stack

```bash
make run
```

### 2. Trigger reconciliation (sample)

#### Sample CSV Data - exist in current repo `/data`

##### Bank Statement CSV (`bank_statements.csv`)

```csv
unique_identifier,amount,date
BRX001,10000,2024-06-01
BRX002,-20000,2024-06-01
BRX004,5000,2024-06-01
```

##### Transactions CSV (`transactions.csv`)
```csv
trxID,amount,type,transactionTime
TRX001,10000,CREDIT,2024-06-01T08:00:00Z
TRX002,20000,DEBIT,2024-06-01T08:15:00Z
TRX003,15000,CREDIT,2024-06-01T08:30:00Z
```

#### CLI Request
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

#### Expected Response
```json
{
  "status": "success",
  "data": {
    "id": 810,
    "reconciliation_type": 1,
    "total_main_row": 0,
    "current_main_row": 0,
    "process_info": "{\"start_time\":1717200000,\"end_time\":1717286399}",
    "status": 1,
    "result": "",
    "create_time": 1748798512,
    "create_by": "radhian",
    "update_time": 1748798512,
    "update_by": "radhian"
  }
}
```

### 3. Get reconciliation result

##### CLI Request
```bash
curl "http://localhost:8080/get_result?log_id=810"
```

#### Expected Response

```json
{
  "status": "success",
  "data": {
    "id": 810,
    "reconciliation_type": 1,
    "total_main_row": 3,
    "current_main_row": 3,
    "process_info": "{\"start_time\":1717200000,\"end_time\":1717286399}",
    "status": 3,
    "result": "{\"total_processed\":3,\"matched\":2,\"unmatched\":1,\"system_unmatched\":[{\"TrxID\":\"TRX003\",\"Amount\":15000,\"Type\":\"CREDIT\",\"TransactionTime\":\"2024-06-01T08:30:00Z\"}],\"bank_unmatched_by_source\":{\"1748798512066907802_bank_statement.csv\":[{\"UniqueIdentifier\":\"BRX004\",\"Amount\":5000,\"Date\":\"2024-06-01T00:00:00Z\"}]},\"total_discrepancy\":20000}",
    "create_time": 1748798512,
    "create_by": "radhian",
    "update_time": 1748798512,
    "update_by": "system"
  }
}
```

#### Result

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

### 4. [Extra] Try large CSVs

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

#### 3. In-Memory Cache Without External Visibility
The current locking mechanism is implemented using in-memory cache, which lacks external observability.

* ⚠️ There is no visibility for debugging or monitoring the lock state across processes or nodes.
Future Improvement: Replace in-memory cache with a distributed cache like Redis to enable robust locking, visibility, and fault tolerance.
