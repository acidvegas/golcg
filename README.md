# golcg
Linear Congruential Generator for scanning IP ranges in Go.

## Features
* Memory efficient IP range scanning
* Deterministic random IP generation
* Sharding support for distributed scanning
* State saving for resuming scans
* Pure Go implementation

## Installation 
```bash
go install github.com/acidvegas/golcg
```

## Usage
```bash
golcg -cidr CIDR [-shard-num N] [-total-shards N] [-seed N] [-state STATE]
```

## Arguments
| Argument        | Description                   |
| --------------- | ----------------------------- |
| `-cidr`         | The CIDR range to scan        |
| `-shard-num`    | The shard number to generate  |
| `-total-shards` | The total number of shards    |
| `-seed`         | The seed for the LCG          |
| `-state`        | The state file to resume from |

## Examples
```bash
# Scan entire IPv4 space
golcg -cidr 0.0.0.0/0

# Scan private ranges
golcg -cidr 10.0.0.0/8
golcg -cidr 172.16.0.0/12
golcg -cidr 192.168.0.0/16

# Distributed scanning (2 shards)
golcg -cidr 0.0.0.0/0 -shard-num 1 -total-shards 2 # One machine
golcg -cidr 0.0.0.0/0 -shard-num 2 -total-shards 2 # Second machine
```

## State Management & Resume Capability

golcg automatically saves its state every 1000 IPs processed to enable resume functionality in case of interruption. The state is saved to a temporary file in your system's temp directory (usually `/tmp` on Unix systems or `%TEMP%` on Windows).

The state file follows the naming pattern:
```
golcg_[seed]_[cidr]_[shard]_[total].state
```

For example:
```
golcg_12345_192.168.0.0_16_1_4.state
```

The state is saved in memory-mapped temporary storage to minimize disk I/O and improve performance. To resume from a previous state:

1. Locate your state file in the temp directory
2. Read the state value from the file
3. Use the same parameters (CIDR, seed, shard settings) with the `--state` parameter

Example of resuming:
```bash
# Read the last state
state=$(cat /tmp/golcg_12345_192.168.0.0_16_1_4.state)

# Resume processing
golcg 192.168.0.0/16 --shard-num 1 --total-shards 4 --seed 12345 --state $state
```

Note: When using the `--state` parameter, you must provide the same `--seed` that was used in the original run.

## How It Works

### IP Address Integer Representation

Every IPv4 address is fundamentally a 32-bit number. For example, the IP address "192.168.1.1" can be broken down into its octets (192, 168, 1, 1) and converted to a single integer:
```
192.168.1.1 = (192 × 256³) + (168 × 256²) + (1 × 256¹) + (1 × 256⁰)
             = 3232235777
```

This integer representation allows us to treat IP ranges as simple number sequences. A CIDR block like "192.168.0.0/16" becomes a continuous range of integers:
- Start: 192.168.0.0   → 3232235520
- End:   192.168.255.255 → 3232301055

By working with these integer representations, we can perform efficient mathematical operations on IP addresses without the overhead of string manipulation or complex data structures. This is where the Linear Congruential Generator comes into play.

### Linear Congruential Generator

golcg uses an optimized LCG implementation with three carefully chosen parameters that work together to generate high-quality pseudo-random sequences:

| Name       | Variable | Value        |
|------------|----------|--------------|
| Multiplier | `a`      | `1664525`    |
| Increment  | `c`      | `1013904223` |
| Modulus    | `m`      | `2^32`       |

###### Modulus
The modulus value of `2^32` serves as both a mathematical and performance optimization choice. It perfectly matches the CPU's word size, allowing for extremely efficient modulo operations through simple bitwise AND operations. This choice means that all calculations stay within the natural bounds of CPU arithmetic while still providing a large enough period for even the biggest IP ranges we might encounter.

###### Multiplier
The multiplier value of `1664525` was originally discovered through extensive mathematical analysis for the Numerical Recipes library. It satisfies the Hull-Dobell theorem's strict requirements for maximum period length in power-of-2 modulus LCGs, being both relatively prime to the modulus and one more than a multiple of 4. This specific value also performs exceptionally well in spectral tests, ensuring good distribution properties across the entire range while being small enough to avoid intermediate overflow in 32-bit arithmetic.

###### Increment
The increment value of `1013904223` is a carefully selected prime number that completes our parameter trio. When combined with our chosen multiplier and modulus, it ensures optimal bit mixing throughout the sequence and helps eliminate common LCG issues like short cycles or poor distribution. This specific value was selected after extensive testing showed it produced excellent statistical properties and passed rigorous spectral tests for dimensional distribution.

### Applying LCG to IP Addresses

Once we have our IP addresses as integers, the LCG is used to generate a pseudo-random sequence that permutes through all possible values in our IP range:

1. For a given IP range *(start_ip, end_ip)*, we calculate the range size: `range_size = end_ip - start_ip + 1`

2. The LCG generates a sequence using the formula: `X_{n+1} = (a * X_n + c) mod m`

3. To map this sequence back to valid IPs in our range:
   - Generate the next LCG value
   - Take modulo of the value with range_size to get an offset: `offset = lcg_value % range_size`
   - Add this offset to start_ip: `ip = start_ip + offset`

This process ensures that:
- Every IP in the range is visited exactly once
- The sequence appears random but is deterministic
- We maintain constant memory usage regardless of range size
- The same seed always produces the same sequence

### Sharding Algorithm

The sharding system employs an interleaved approach that ensures even distribution of work across multiple machines while maintaining randomness. Each shard operates independently using a deterministic sequence derived from the base seed plus the shard index. The system distributes IPs across shards using modulo arithmetic, ensuring that each IP is assigned to exactly one shard. This approach prevents sequential scanning patterns while guaranteeing complete coverage of the IP range. The result is a system that can efficiently parallelize work across any number of machines while maintaining the pseudo-random ordering that's crucial for network scanning applications.

---

###### Mirrors: [acid.vegas](https://git.acid.vegas/golcg) • [SuperNETs](https://git.supernets.org/acidvegas/golcg) • [GitHub](https://github.com/acidvegas/golcg) • [GitLab](https://gitlab.com/acidvegas/golcg) • [Codeberg](https://codeberg.org/acidvegas/golcg)