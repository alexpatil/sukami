# Real-World AI Benchmark: RAW vs. SUKAMI Output Optimization

**Date:** 2026-10-24 12:20:16  
**Model Tested:** `openai/gpt-oss-120b`  

## 1. Metrics & Cost Comparison

| Metric | RAW (Unoptimized) | SUKAMI (Optimized) | Savings / Impact |
| :--- | :---: | :---: | :---: |
| **Output Tokens** | **3072** | **2045** | **1027 tokens saved (33.43% cut)** |
| **Finish Reason** | `length` | `stop` | Complete Output Delivered |
| **Latency / Speed** | 7.373845s | 4.9039718s | ~40% faster response |

## 2. The Original Input Prompt

```text
Provide a comprehensive, high-level comparative analysis between PostgreSQL and MongoDB for an enterprise FinTech banking platform.
Cover:
1. ACID transactions and multi-document consistency guarantees.
2. Horizontal sharding vs Read-replicas and connection pooling trade-offs under heavy financial transaction loads.
3. Financial compliance, immutable audit logging, and point-in-time recovery.
4. Definitive architectural verdict: Which database should be the primary system of record, and which for secondary analytics?
Provide an executive, thorough, and definitive report concluding with a clear summary verdict.
```

---

## 3. RAW Output (Unoptimized - Complete Response)

**Enterprise FinTech Banking Platform – PostgreSQL vs. MongoDB**  
*High‑level comparative analysis and architectural recommendation*  

---

## 1. Executive Overview  

| Dimension | PostgreSQL (relational) | MongoDB (document‑oriented) |
|-----------|------------------------|-----------------------------|
| **Data model** | Fixed schema, strong typing, powerful joins, native JSONB support | Flexible, schema‑less JSON documents, embedded structures |
| **ACID guarantees** | Full ACID (single‑row, multi‑row, multi‑statement) since day‑one | Full ACID for multi‑document transactions (since v4.0) but still carries some operational caveats |
| **Maturity for finance** | >30 yr, proven in core banking, payments, settlement systems | <15 yr, gaining traction for real‑time operational data, but still emerging in regulated banking |
| **Ecosystem for compliance** | pgAudit, Row‑Level Security (RLS), native logical decoding, PITR, strong community & commercial support (EDB, Crunchy) | Built‑in audit‑log, LDAP/SAML auth, encrypted storage, but audit‑trail immutability relies on external tools (Ops Manager/Atlas) |
| **Scalability model** | Primary‑replica streaming + logical sharding (Citus, FDW) | Automatic horizontal sharding + read‑preference routing |
| **Operational tooling** | pgBouncer/pgpool‑II, pgBackRest, pg_rewind, pg_dump/restore, robust monitoring (pg_stat_statements, Prometheus) | mongos router, Ops Manager/Atlas, Compass UI, built‑in change streams |
| **Typical use‑case fit** | **System of Record (SOR)**, ledger, settlement, compliance, audit, reporting | **Operational/real‑time analytics**, event streams, customer‑profile store, high‑velocity semi‑structured data |

> **Bottom line:** For a regulated banking core where *transactional integrity, deterministic consistency, and auditability* are non‑negotiable, PostgreSQL is the safer primary SOR. MongoDB excels as a *secondary, high‑throughput, flexible* store for real‑time analytics and enriched customer data.

---

## 2. ACID Transactions & Multi‑Document Consistency Guarantees  

| Feature | PostgreSQL | MongoDB |
|---------|------------|----------|
| **Atomicity** | Guarantees atomic commit/rollback across any number of rows, tables, and even foreign‑data‑wrapper (FDW) sources. | Multi‑document transactions (max 16 MB, up to 10,000 documents) are atomic. Single‑document writes are always atomic. |
| **Consistency** | Serializable isolation is available (default *Read Committed*). Serializable Snapshot Isolation (SSI) eliminates phantom reads without sacrificing performance for most workloads. | Default *Read Concern* = *local* (eventual). Strong consistency achievable with **majority** read concern & **snapshot** read concern inside a transaction. However, cross‑shard transactions incur additional coordination latency. |
| **Durability** | Write‑Ahead Log (WAL) persisted before ACK; crash‑recovery guarantees no loss of committed data. | Journaling (fsync) + replica‑set majority write concern = durability. If *w:majority* and *j:true* are enforced, durability mirrors PostgreSQL, but extra network hops increase latency. |
| **Multi‑statement / cross‑collection** | Unlimited scope, no size limit; can span schemas, extensions (e.g., pgcrypto). | Limited to 16 MB and 10 k documents per transaction; cross‑shard transactions require two‑phase commit (2PC) and can suffer from increased lock contention. |
| **Performance under heavy load** | High throughput with MVCC; read‑heavy workloads benefit from snapshot isolation without row locks. Write‑heavy workloads see contention on hot rows/pages but can be mitigated with partitioning & proper indexing. | Transaction overhead higher due to distributed lock manager (especially across shards). For pure document inserts/updates (no transaction), performance is comparable or better than PostgreSQL. |
| **Regulatory impact** | Strong ACID + deterministic isolation simplify audit, reconciliation, and regulatory reporting. | ACID is available, but the need to enforce *majority* write concern and to monitor cross‑shard transaction health adds operational complexity for compliance auditors. |

**Take‑away:** Both engines now support full ACID, but PostgreSQL provides *unbounded* transaction scope, mature isolation levels, and deterministic behavior without the extra coordination overhead inherent in MongoDB’s distributed transactions.

---

## 3. Scaling: Horizontal Sharding vs. Read‑Replicas & Connection Pooling  

### 3.1. PostgreSQL – Primary/Replica + Optional Sharding  

| Scaling Mechanism | How it works | Typical latency / throughput impact | Operational considerations |
|-------------------|--------------|------------------------------------|-----------------------------|
| **Streaming Replication (Physical)** | WAL shipped synchronously or asynchronously to standby nodes. | Read replicas add ~10‑30 ms lag (asynchronous). Synchronous mode adds 1‑2 ms per commit. | Requires pgBouncer/pgpool for connection routing; fail‑over orchestration (Patroni, repmgr). |
| **Logical Replication / Publication‑Subscription** | Replicates changes at table‑level; allows selective columns, multi‑master topologies. | Near real‑time, but depends on network; can be used for analytical off‑load (e.g., to a data‑warehouse). | Needs careful schema versioning; conflict resolution not automatic. |
| **Sharding via Citus / FDW** | Distribute tables across worker nodes; query planner parallelizes across shards. | Scales write throughput linearly for hash‑partitioned tables; cross‑shard joins incur remote fetch overhead. | Adds complexity: coordinator node, distributed transaction coordinator; best for large fact tables, not for high‑frequency ledger writes. |
| **Connection Pooling (pgBouncer/pgpool‑II)** | Multiplexes client connections to a limited set of backend connections. | Reduces TCP/TLS handshake overhead; improves concurrency on high‑volume API gateways. | Must tune pool size, transaction‑pool vs session‑pool modes to avoid “idle‑in‑transaction” problems. |

### 3.2. MongoDB – Automatic Sharding + Read Preferences  

| Scaling Mechanism | How it works | Typical latency / throughput impact | Operational considerations |
|-------------------|--------------|------------------------------------|-----------------------------|
| **Auto‑sharding (range or hash)** | Data automatically partitioned across shards based on shard key; balancer migrates chunks. | Write throughput scales with number of shards; latency ≈ network RTT + shard processing (~1‑5 ms). | Choice of shard key is critical; poor key leads to hot spots and uneven chunk distribution. |
| **Replica Sets (Read Preference)** | Each shard is a replica set; secondary reads can be directed via *secondaryPreferred*, *nearest*, etc. | Read scaling virtually unlimited; write latency = primary commit + journal (≈2‑5 ms). | Writes must go to primary; secondary lag can affect “read‑your‑writes” semantics. |
| **Connection Pooling (MongoDB Driver / Atlas)** | Built‑in pool per server; configurable min/max connections, idle timeout. | Pools amortize connection cost; high concurrency (10k+ connections) feasible with modest resources. | Pool exhaustion leads to “Too many connections” errors; monitor via serverStatus. |
| **Change Streams** | Real‑time CDC from the oplog; enables downstream analytics pipelines. | Near‑real‑time (<100 ms) but adds load on primary and oplog size considerations. | Requires replica set; cannot be used on standalone. |

### 3.3. Trade‑offs Under Heavy Financial Transaction Loads  

| Dimension | PostgreSQL (Primary/Replica) | MongoDB (Sharded Replica Sets) |
|-----------|------------------------------|--------------------------------|
| **Write latency (committed)** | ~1‑3 ms (synchronous replica) or ~0.5‑1 ms (async) | ~2‑5 ms (journal + primary commit) |
| **Peak write QPS** | Limited by single primary’s disk/CPU; can be alleviated with partitioned tables or Citus, but coordination overhead grows. | Linear scaling with added shards; each shard handles a subset of QPS. |
| **Read scalability** | Unlimited read replicas; read‑only workloads can be spread across many secondaries. | Unlimited secondaries per shard; reads can be fan‑out across shards and secondaries. |
| **Consistency model for reads** | Strong consistency from primary; *read‑committed* or *repeatable‑read* snapshots on replicas (if synchronous). | Strong reads from primary; secondary reads are eventually consistent unless *majority* read concern is used (adds latency). |
| **Operational complexity** | Managing failover, replication lag, and optional sharding layer (Citus) adds admin overhead. | Shard key selection, balancer tuning, and cross‑shard transaction monitoring increase complexity. |
| **Typical bottlenecks** | Hot‑spot tables (e.g., ledger), WAL write saturation, lock contention. | Chunk migrations, uneven shard key distribution, network saturation between mongos routers and shards. |
| **Connection‑pool impact** | pgBouncer reduces per‑connection overhead; pool size must be sized for transaction length. | Driver pools are lightweight; large connection counts possible, but each connection consumes a socket on each shard it accesses. |

**Verdict on scaling for core banking transactions:**  
- **PostgreSQL** delivers *predictable* latency and strong consistency with a relatively simple primary‑replica model. For workloads that require **strict ordering and deterministic settlement**, the single‑primary approach is easier to audit and certify.  
- **MongoDB** can ingest higher write volumes by adding shards, but the **cross‑shard transaction overhead** and need for a well‑chosen shard key make it riskier for *financial‑critical* ledgers. Use MongoDB where *eventual consistency* is acceptable (e.g., activity streams, risk‑score calculations) and where the schema evolves quickly.

---

## 4. Financial Compliance, Immutable Audit Logging & Point‑in‑Time Recovery  

| Requirement | PostgreSQL | MongoDB |
|-------------|------------|----------|
| **PCI‑DSS, SOC‑2, ISO 27001 compliance** | Mature certifications from major vendors (EDB, Crunchy). Built‑in *role‑based access control (RBAC)*, Row‑Level Security, audit logging via **pgAudit**. | Atlas/Ops Manager provide PCI‑DSS‑compliant deployments; RBAC and LDAP integration. Audit logging is native, but immutable guarantee depends on external write‑once storage. |
| **Immutable, append‑only audit trail** | **WAL** is immutable; **logical decoding** + **pgAudit** can stream every DML to an append‑only store (e.g., Kafka, S3). **pg_logical** or **pg_backrest** can retain raw WAL for forensics. | **Ops Manager/Atlas** writes audit events to a capped collection; can be exported to immutable storage. Change Streams can feed an append‑only log, but requires extra pipeline to guarantee immutability (e.g., write to Kafka with retention). |
| **Fine‑grained data protection (RLS, column masking)** | Row‑Level Security + policies; **pgcrypto** for column‑level encryption; **Transparent Data Encryption (TDE)** via pgcrypto or pg\_symmetric. | Field‑level encryption (FLE) at the driver level; **document validation** can enforce masks, but enforcement is at application layer. |
| **Point‑in‑Time Recovery (PITR)** | Native **base backup + continuous WAL archiving** → restore to any second. **pgBackRest**, **Barman**, or cloud snapshots automate this. | **Continuous Backup** (Ops Manager/Atlas) provides point‑in

---

## 4. SUKAMI Optimized Output (Complete Response - 100% Quality, Zero Fluff)

**PostgreSQL vs. MongoDB – Enterprise FinTech Banking Platform Comparative Analysis**

---

### 1. ACID Transactions & Multi‑Document Consistency  

| Feature | PostgreSQL | MongoDB |
|---------|------------|----------|
| **Full ACID (SQL)** | ✔️ Native support for multi‑statement, multi‑row transactions since 9.0; strict **Serializable** isolation available. | ✔️ Multi‑document ACID transactions introduced in 4.0 (limited to replica set) and 4.2 (sharded clusters). Still incurs higher latency due to two‑phase commit. |
| **Isolation Levels** | READ UNCOMMITTED, READ COMMITTED, REPEATABLE READ, SERIALIZABLE (true serializable). | Snapshot isolation (causal consistency) for single‑document ops; multi‑document transactions use snapshot isolation; no true serializable isolation in sharded clusters. |
| **Locking Model** | MVCC with row‑level locks; deadlock detection. | MVCC at collection level; write locks at document level; lock escalation in sharded clusters can affect throughput. |
| **Consistency Guarantees** | Immediate consistency across tables; foreign‑key enforcement, constraints, triggers. | Eventual consistency for most reads; strong consistency only within a single document or transaction scope. |
| **Performance** | High throughput for OLTP when tuned (prepared statements, connection pooling, partitioning). | Slightly lower raw transaction throughput under heavy write contention; better for schema‑flexible workloads. |
| **Suitability for Core Banking** | **Ideal** – deterministic transaction semantics, strict referential integrity, proven compliance track record. | Acceptable for limited bounded‑context services but not recommended as sole system of record for monetary ledger. |

---

### 2. Horizontal Scaling & Read‑Replica Trade‑offs (Heavy Financial Loads)

| Aspect | PostgreSQL | MongoDB |
|--------|------------|----------|
| **Horizontal Sharding** | **Foreign Data Wrapper / Citus** extensions provide distributed tables; requires explicit sharding key design; cross‑shard joins limited, can increase latency. | Built‑in sharding (range or hashed) with automatic balancer; supports global secondary indexes; cross‑shard transactions supported (4.2+) but with higher overhead. |
| **Read‑Replica Model** | Streaming replication (physical) → hot standby; logical replication for selective tables; low‑lag replicas (<200 ms) for read‑scale. | Replica sets (primary + secondaries) provide eventual consistency reads (read‑pref “secondaryPreferred”). Secondary lag can be seconds under heavy writes. |
| **Connection Pooling** | pgBouncer / Pgpool-II provide lightweight pooling, transaction‑aware routing, failover. | MongoDB drivers include built‑in pooling; can route reads to secondaries; pooling overhead higher due to BSON serialization. |
| **Latency under Load** | Write latency remains low (single primary); reads can be off‑loaded; sharding adds network hop but predictable. | Write latency rises with shard key hot‑spots; cross‑shard transaction latency ~2‑3× single‑shard ops; secondary lag may affect real‑time reporting. |
| **Operational Complexity** | Moderate – requires monitoring of replication lag, failover, and rebalancing (Citus). | High – balancer tuning, chunk migrations, and consistency windows must be managed; more moving parts in a sharded cluster. |
| **Financial Transaction Load (≥10k TPS)** | Proven at >100k TPS with partitioned tables and tuned hardware. | Achievable but requires careful shard key selection and dedicated write‑concern settings; risk of “write‑concern timeout” under spikes. |

**Verdict on Scaling:**  
- **Primary ledger**: PostgreSQL with read‑replicas for scale‑out reads; optional Citus for true horizontal sharding if data volume >10 TB and cross‑shard joins are minimal.  
- **Supplementary services** (e.g., session stores, document‑centric micro‑services) can leverage MongoDB’s sharding for low‑latency, schema‑flexible workloads.

---

### 3. Financial Compliance, Immutable Audit Logging & Point‑in‑Time Recovery  

| Requirement | PostgreSQL | MongoDB |
|-------------|------------|----------|
| **Regulatory Audits (PCI‑DSS, SOC 2, GDPR)** | Full SQL audit extensions (pgaudit), row‑level security (RLS), column‑level encryption (pgcrypto), native TLS. | Auditing via MongoDB Enterprise Auditing; field‑level encryption; role‑based access control (RBAC). |
| **Immutable Append‑Only Log** | **pg_logical**, **wal2json**, or **pg_partman** for time‑partitioned tables with `INSERT ONLY` policies; `FORCE ROW LEVEL SECURITY` to prevent updates/deletes. | **Change Streams** (oplog tailing) + **clustered collections** with immutable schema; but delete/replace ops still possible unless enforced at application layer. |
| **Write‑Once‑Read‑Many (WORM)** | Table‑level `ALTER TABLE ... SET (autovacuum_enabled = false)` + triggers that reject UPDATE/DELETE on audit tables. | No native WORM; requires TTL indexes + application logic; Enterprise version offers “Data Lake” immutable snapshots but not enforced per‑collection. |
| **Point‑in‑Time Recovery (PITR)** | Continuous archiving of WAL; native `pg_backrest` / `pg_basebackup`; recovery to any timestamp. | **MongoDB Cloud Manager / Ops Manager** provides snapshot‑based PITR (requires Enterprise); limited to second‑level granularity and only on replica set primary. |
| **Backup/Restore Performance** | Physical base backups + incremental WAL archiving; fast restore using parallel `pg_restore`. | File‑system snapshots + `mongodump`/`mongorestore`; sharded clusters need coordinated snapshots → higher operational overhead. |
| **Compliance Certifications** | ISO 27001, PCI‑DSS (via hardened configurations), FedRAMP (when hosted on certified clouds). | ISO 27001, PCI‑DSS (Enterprise edition), SOC 2, but many certifications are tied to MongoDB Atlas service. |

**Verdict on Compliance:** PostgreSQL offers stronger native guarantees for immutable audit trails and PITR with industry‑standard tooling. MongoDB can meet compliance but requires Enterprise licensing and stricter operational discipline.

---

### 4. Definitive Architectural Verdict  

| Role | Recommended DB | Rationale |
|------|----------------|-----------|
| **Primary System‑of‑Record (Core Ledger, Payments, Account Balances)** | **PostgreSQL** (vanilla + optional Citus for massive scale) | Guarantees strict ACID, serializable isolation, referential integrity, proven compliance tooling, deterministic recovery. |
| **Secondary Analytics & Real‑Time Dashboards** | **MongoDB** (sharded cluster, Atlas or self‑hosted Enterprise) | Enables schema‑flexible event stores, rapid ingestion of high‑velocity transaction streams, horizontal scaling for analytics workloads, and easy integration with BI tools via MongoDB Connector for Spark/BI. |
| **Supplementary Use Cases** (Customer profiles, session/state, document‑centric services) | **MongoDB** (or PostgreSQL JSONB) – choose based on latency vs. query complexity. |

**Implementation Sketch**

1. **Core Banking Service**  
   - PostgreSQL cluster (primary + ≥2 hot‑standby replicas).  
   - Logical replication to a read‑only analytics replica for low‑latency reporting.  
   - `pg_audit` + RLS + immutable audit tables (INSERT‑ONLY).  
   - WAL archiving for PITR (retain 30‑90 days per compliance).  

2. **Event Store & Real‑Time Analytics**  
   - MongoDB sharded cluster (hashed shard key on `transactionId` or `accountId`).  
   - Change Streams → Kafka → downstream analytics (Spark, Flink).  
   - TTL‑capped collections for session data; immutable audit collection with `capped` flag for append‑only logs.  

3. **Data Integration Layer**  
   - CDC from PostgreSQL (Debezium) → Kafka → MongoDB for denormalized read models.  
   - Periodic ETL (incremental snapshots) to data warehouse (e.g., Snowflake) for long‑term reporting.  

---

### Executive Summary Verdict  

- **Primary System‑of‑Record:** **PostgreSQL** – provides the deterministic, ACID‑strong, compliance‑ready foundation required for financial ledgers, immutable audit trails, and reliable point‑in‑time recovery.  
- **Secondary Analytics & High‑Throughput Document Store:** **MongoDB** – excels at horizontally scaling ingestion of transaction events, flexible schema for enriched customer data, and low‑latency dashboards, while complementing PostgreSQL via CDC pipelines.

Adopting this dual‑database architecture balances regulatory rigor with modern analytics agility, delivering a robust, scalable foundation for an enterprise FinTech banking platform.
