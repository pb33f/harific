# HARific

![logo](harific-logo.png)

We renamed it from braid. HARific is more fun.



| File Size | Entries | Time   | Throughput  | Time/Entry |
|-----------|---------|--------|-------------|------------|
| 700MB     | 9,720   | 1.90s  | 367.76 MB/s | 195.84 μs  |
| 1GB       | 14,126  | 2.77s  | 369.87 MB/s | 196.00 μs  |
| 2GB       | 28,262  | 5.57s  | 367.93 MB/s | 196.96 μs  |
| 5GB       | 70,689  | 13.62s | 375.80 MB/s | 192.73 μs  |

Key Findings:

✅ Consistent Performance: Throughput stays rock-solid at ~370 MB/s regardless of file size
✅ Linear Scaling: Processing time scales perfectly linearly with file size
✅ Predictable: ~195 microseconds per entry consistently
✅ No Degradation: The 5GB file actually performs slightly better than smaller files!

This confirms the V1 implementation is highly optimized and scales beautifully. The consistent 370+ MB/s throughput means:
- A 10GB file would take ~27 seconds
- A 100MB file would take ~270ms

These are our foundational numbers and this is where we start. We have spent a whole bunch of time on this today looking at different
parsing, decoding, chunking and scanning and streaming. We tried everything and this is as fast as we can go accurately. 



Driven by massive frustration with having to try and diagnose a customer's problems from an experience they had through the browser
Whilst using ES. Being unable to see what the customer saw in the way that the customer saw it, trying to diagnose performance problems or 
rendering issues is really hard.

Or at least, it was.. (in theory)

I am going to build a tool that allows us to do a few things.

1. Visually explore gigantic HAR files in the terminal
2. Run a replay server that will replay every response in sequence back to the browser.
3. Set breakpoints on responses and pause the conversation anywhere.

Being able to inspect and replay the HAR file using the UI should allow us to see what a customer saw. 

Will it work? I don't know, but we are going to find out.

