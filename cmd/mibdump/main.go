package main

import (
	"flag"
	"log"
)

func main() {
	log.SetFlags(0) // Disable log prefixes

	// --- Command Line Flags ---
	mibFilePath := flag.String("mibfile", "", "Path to the single MIB file to parse (mutually exclusive with -dir)")
	mibDirPath := flag.String("dir", "", "Path to the directory of MIB files to process recursively (mutually exclusive with -mibfile)")
	outputType := flag.String("output", "all", "Type of output for single file mode: ast, resolved, or all (default)")
	dumpOutput := flag.Bool("dump", false, "Dump the full JSON output instead of a diff summary (single file mode only)")
	flag.Parse()

	// --- Validate Flags ---
	if (*mibFilePath == "" && *mibDirPath == "") || (*mibFilePath != "" && *mibDirPath != "") {
		log.Fatal("Error: Exactly one of -mibfile or -dir must be specified")
	}

	if *mibDirPath != "" && (*outputType != "all" || *dumpOutput) {
		log.Println("Warning: -output and -dump flags are ignored when using -dir mode.")
		// Reset flags to defaults for directory mode to avoid confusion
		*outputType = "all" // Implicitly 'resolved' for comparison
		*dumpOutput = false // Ensure dump is off for dir mode summary
	}

	// --- Dispatch to Processing Functions ---
	if *mibFilePath != "" {
		// Validate output type for single file mode
		if *outputType != "ast" && *outputType != "resolved" && *outputType != "all" {
			log.Fatalf("Error: invalid -output type %q for single file mode. Must be 'ast', 'resolved', or 'all'", *outputType)
		}
		// Call the processing function (now in process.go)
		processSingleMibFile(*mibFilePath, *outputType, *dumpOutput)
	} else {
		// Call the directory processing function (now in process.go)
		processDirectory(*mibDirPath)
	}
}
