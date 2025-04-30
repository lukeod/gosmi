# MIB Dump Comparison Tool

A utility for comparing MIB file parsing and resolution between two implementations of the gosmi library.

## Overview

The MIB Dump tool compares how MIB (Management Information Base) files are parsed and resolved by two different implementations of the gosmi library:

- **Fork**: The updated implementation by lukeod (github.com/lukeod/gosmi)
- **Mainline**: The original implementation by sleepinggenius2 (github.com/sleepinggenius2/gosmi)

This tool is useful for:
- Identifying differences in parsing behavior
- Validating compatibility between implementations
- Debugging MIB parsing issues
- Understanding how changes in the parser affect MIB resolution

## Installation

### Prerequisites

- Go 1.16 or later
- Access to both gosmi implementations

### Building from Source

1. Clone the repository:
   ```
   git clone https://github.com/lukeod/gosmi.git
   cd gosmi
   ```

2. Build the mibdump tool:
   ```
   cd cmd/mibdump
   go build
   ```

## Usage

```
./mibdump -mibfile <path-to-mib-file> [options]
```

### Command-line Options

- `-mibfile <path>`: (Required) Path to the MIB file to parse
- `-output <type>`: Type of output to generate
  - `ast`: Compare only the Abstract Syntax Tree (AST) parsing
  - `resolved`: Compare only the resolved MIB data
  - `all`: Compare both AST and resolved data (default)
- `-dump`: Dump the full JSON output instead of a diff summary

### Output Formats

The tool provides two main types of output:

1. **AST Comparison**: Compares the raw parsed structure of the MIB file
2. **Semantic Comparison**: Compares the resolved MIB data, including:
   - Module information differences
   - Added/removed/modified nodes
   - Added/removed/modified types

## Examples

### Basic Comparison

Compare a MIB file using both implementations:

```
./mibdump -mibfile /path/to/EXAMPLE-MIB.mib
```

Example output:
```
--- Comparison Results ---
Performing semantic comparison of resolved MIB data...
✅ No semantic differences found in resolved MIB data.
   Compared: 10 Nodes, 5 Types
```

### AST-Only Comparison

Compare only the AST parsing:

```
./mibdump -mibfile /path/to/EXAMPLE-MIB.mib -output ast
```

Example output:
```
--- Comparison Results ---
Comparing ASTs (or showing errors if resolution failed)...
✅ No differences found in AST or parsing errors.
```

### Resolved Data Comparison

Compare only the resolved MIB data:

```
./mibdump -mibfile /path/to/EXAMPLE-MIB.mib -output resolved
```

Example output:
```
--- Comparison Results ---
Performing semantic comparison of resolved MIB data...
⚠️ Semantic differences found in resolved MIB data:
{
  "nodesAdded": [
    {
      "name": "exampleCounter",
      "oid": "1.3.6.1.4.1.12345.1.1",
      "kind": "Scalar"
    }
  ],
  "nodesRemoved": [],
  "nodesCompared": 10,
  "nodesOk": 9,
  "nodesAddedCount": 1,
  "nodesRemovedCount": 0,
  "nodesModifiedCount": 0
}
```

### Full JSON Dump

Dump the full JSON output for detailed analysis:

```
./mibdump -mibfile /path/to/EXAMPLE-MIB.mib -dump
```

This will output a comprehensive JSON structure containing all the parsed and resolved data from both implementations.

## Interpreting Results

### No Differences

When you see `✅ No semantic differences found in resolved MIB data.`, it means both implementations parse and resolve the MIB file identically.

### Semantic Differences

When differences are found, they are categorized as:

1. **Added Nodes/Types**: Elements present in the fork implementation but not in the mainline
2. **Removed Nodes/Types**: Elements present in the mainline implementation but not in the fork
3. **Modified Nodes/Types**: Elements present in both implementations but with different attributes

### Common Differences

Some common differences you might encounter:

- **OID Resolution**: Different OID assignments for the same named nodes
- **Type Resolution**: Different base type assignments
- **Node Kind**: Different node kind classifications (e.g., Scalar vs. Column)
- **Missing Nodes**: Nodes present in one implementation but missing in the other

## Troubleshooting

### Parse Errors

If you see parse errors like:
```
[Fork] Error parsing AST: unexpected token "END" (expected "DEFINITIONS")
```

This indicates a syntax error in the MIB file or an issue with the parser.

### Resolution Errors

If you see resolution errors like:
```
[Fork] Error loading/resolving target MIB: unknown module
```

This usually means:
- The MIB file has dependencies that aren't available
- The MIB name doesn't match the filename
- There are import errors in the MIB file

### Dependency Issues

Many MIB files depend on standard MIBs like SNMPv2-SMI, SNMPv2-TC, etc. Make sure these are available in the same directory as your target MIB file.

## Advanced Usage

### Comparing Multiple MIBs

To compare multiple MIB files, you can use a shell script:

```bash
#!/bin/bash
for mib in /path/to/mibs/*.mib; do
  echo "Comparing $mib"
  ./mibdump -mibfile "$mib"
  echo "------------------------"
done
```

### Filtering Results

You can pipe the output through tools like `jq` to filter specific differences:

```bash
./mibdump -mibfile /path/to/EXAMPLE-MIB.mib -dump | jq '.fork_results.resolved.nodes'
```

## Contributing

Contributions to improve the mibdump tool are welcome. Please submit pull requests or open issues on the GitHub repository.