# HARific - High-Performance HAR File Toolkit

![logo](harific-logo.png)

## Quick Start

### Installation

```bash
git clone https://github.com/pb33f/harific.git
cd harific
make build
```

### Usage

```bash
# View HAR file in interactive TUI
./bin/harific view recording.har

# Generate test HAR files
./bin/harific generate -n 1000 -o test.har

# Generate with search term injection
./bin/harific generate -n 100 --inject apple,banana --locations url,response.body

# Show all commands
./bin/harific --help
```

## Background

Driven by frustration with diagnosing customer problems from browser experiences, HARific was built to solve a real problem: being unable to see what the customer saw, in the way they saw it. Diagnosing performance problems or rendering issues without proper tools is really hard. HARific provides visual exploration of gigantic HAR files in the terminal, with plans for a replay server that will replay every response back to the browser, complete with breakpoints to pause the conversation anywhere.

## Architecture

### Indexing

```mermaid
flowchart TB
    HAR[5GB HAR File] --> Index[One-time Index Build<br/>~10MB memory]
    Index --> Offset[Offset Map<br/>O1 lookup]

    User[User Request Entry #5000] --> Offset
    Offset --> Seek[Direct Seek to Byte Position<br/>No scanning]
    Seek --> Read[Read Only Entry #5000<br/>~20KB]
    Read --> Display[Display to User]

    style HAR fill:#fff4e1
    style Index fill:#e1ffe1
    style Read fill:#e1f5ff

    Note1[Memory Used: ~57MB<br/>for 1.3GB file]
    Note2[Speed: ~370 MB/s<br/>consistent throughput]
```

### HAR Streamer Engine Architecture

```mermaid
flowchart LR
    subgraph "Index Phase (One-time)"
        File[HAR File] --> Scanner[Line Scanner]
        Scanner --> Detector[Entry Detector<br/>comma,brace counting]
        Detector --> Builder[Index Builder]
        Builder --> |xxHash| Dedup[String Interning<br/>256 shards]
        Dedup --> Index[(Index<br/>10MB for 10K entries)]
    end

    subgraph "Access Phase (Per Request)"
        Request[Get Entry N] --> Lookup[O1 Map Lookup]
        Lookup --> |offset,length| Reader[Entry Reader]
        Reader --> Seek[File Seek]
        Seek --> Parse[JSON Parse<br/>Single Entry Only]
        Parse --> Entry[Entry Object]
    end

    style File fill:#fff4e1
    style Index fill:#e1ffe1
    style Entry fill:#e1f5ff
```

### Search Engine Architecture

```mermaid
flowchart TB
    User[Search Query] --> Compile[Compile Pattern Once]
    Compile --> Partition[Partition Work<br/>10K entries → 100 batches]

    Partition --> Queue[Work Queue Channel]
    Queue --> W1[Worker 1]
    Queue --> W2[Worker 2]
    Queue --> W8[Worker 8]

    W1 --> Phase1[Phase 1: Metadata<br/>URL, Method, Status<br/>NO DISK I/O]
    W2 --> Phase1
    W8 --> Phase1

    Phase1 --> |No Match| Phase2[Phase 2: Load Entry<br/>Read from Disk]
    Phase1 --> |Match| Result[Return Result]

    Phase2 --> Search[Search Headers,<br/>Body, Cookies]
    Search --> Result

    Result --> Channel[Results Channel]
    Channel --> User

    subgraph "Resource Pools"
        BP[Buffer Pool<br/>64KB × 8]
        FP[File Handle Pool<br/>8 handles]
    end

    W1 -.-> BP
    W2 -.-> BP
    W8 -.-> BP
    W1 -.-> FP
    W2 -.-> FP
    W8 -.-> FP

    style Phase1 fill:#e1ffe1
    style Phase2 fill:#ffe1e1
    style BP fill:#e1f5ff
    style FP fill:#e1f5ff
```

## Performance

| File Size | Entries | Time   | Throughput  | Time/Entry |
|-----------|---------|--------|-------------|------------|
| 700MB     | 9,720   | 1.90s  | 367.76 MB/s | 195.84 μs  |
| 1GB       | 14,126  | 2.77s  | 369.87 MB/s | 196.00 μs  |
| 2GB       | 28,262  | 5.57s  | 367.93 MB/s | 196.96 μs  |
| 5GB       | 70,689  | 13.62s | 375.80 MB/s | 192.73 μs  |

- **Consistent Performance**: ~370 MB/s regardless of file size
- **Linear Scaling**: Processing time scales perfectly with file size
- **Predictable**: ~195 microseconds per entry consistently
- **Memory Efficient**: ~57MB for 1.3GB file 
