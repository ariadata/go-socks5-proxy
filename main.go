package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/armon/go-socks5"
)

// Version will be set during build time
var Version = "dev"

type arrayFlags []string

func (i *arrayFlags) String() string {
	return strings.Join(*i, ", ")
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func printHelp() {
	fmt.Printf("SOCKS5 Proxy Server v%s\n", Version)
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  ./socks5-server [OPTIONS]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --host HOST              Host to bind to (default: 0.0.0.0)")
	fmt.Println("  --port PORT              Port to listen on (default: 1080)")
	fmt.Println("  --users FILE             Path to users configuration file")
	fmt.Println("  --user USER              User credentials in format username:password (can be used multiple times)")
	fmt.Println("  --allow-internal         Allow proxying to private/internal IPs (default: false)")
	fmt.Println("  --version                Show version information")
	fmt.Println("  --help                   Show this help message")
	fmt.Println()
	fmt.Println("Authentication:")
	fmt.Println("  - If no users are configured, the server runs without authentication")
	fmt.Println("  - If at least one user is configured, authentication is required")
	fmt.Println("  - Users can be loaded from file and/or added via multiple --user flags")
	fmt.Println()
	fmt.Println("Internal destinations (blocked unless --allow-internal):")
	fmt.Println("  - Loopback, RFC1918 private, link-local, unspecified, CGNAT 100.64.0.0/10, ULA IPv6")
}

func addUser(credentials socks5.StaticCredentials, username, password string) {
	// Check if user already exists with different password
	if existingPassword, exists := credentials[username]; exists {
		// If same password, skip
		if existingPassword != password {
			// Create a new entry for same user with different password
			counter := 1
			for {
				newKey := fmt.Sprintf("%s_%d", username, counter)
				if _, exists := credentials[newKey]; !exists {
					credentials[newKey] = password
					break
				}
				counter++
			}
		}
	} else {
		credentials[username] = password
	}
}

func main() {
	var (
		port          = flag.Int("port", 1080, "Port to listen on")
		host          = flag.String("host", "0.0.0.0", "Host to bind to")
		usersFile     = flag.String("users", "", "Path to users configuration file")
		allowInternal = flag.Bool("allow-internal", false, "Allow proxying to private/internal IPs (default: false)")
		help          = flag.Bool("help", false, "Show help message")
		version       = flag.Bool("version", false, "Show version information")
	)

	var userFlags arrayFlags
	flag.Var(&userFlags, "user", "User credentials in format username:password (can be used multiple times)")

	flag.Parse()

	// Check for version flag
	if *version {
		fmt.Printf("SOCKS5 Proxy Server v%s\n", Version)
		os.Exit(0)
	}

	// Check for help flag
	if *help {
		printHelp()
		os.Exit(0)
	}

	// Create a map to store credentials
	credentials := make(socks5.StaticCredentials)

	// Load credentials from file if provided
	if *usersFile != "" {
		if _, err := os.Stat(*usersFile); err == nil {
			file, err := os.Open(*usersFile)
			if err != nil {
				log.Printf("Warning: Could not open users file %s: %v", *usersFile, err)
			} else {
				defer file.Close()

				scanner := bufio.NewScanner(file)
				for scanner.Scan() {
					line := strings.TrimSpace(scanner.Text())
					if line == "" || strings.HasPrefix(line, "#") {
						continue // Skip empty lines and comments
					}
					parts := strings.Split(line, ":")
					if len(parts) != 2 {
						log.Printf("Warning: Invalid auth-pair in %s: %s", *usersFile, line)
						continue
					}
					addUser(credentials, parts[0], parts[1])
				}

				if err := scanner.Err(); err != nil {
					log.Printf("Warning: Error reading users file: %v", err)
				}
			}
		} else {
			log.Printf("Warning: Users file %s does not exist", *usersFile)
		}
	}

	// Add users from --user flags
	for _, userFlag := range userFlags {
		parts := strings.Split(userFlag, ":")
		if len(parts) != 2 {
			log.Fatalf("Invalid user format: %s (expected username:password)", userFlag)
		}
		addUser(credentials, parts[0], parts[1])
	}

	// Create SOCKS5 server configuration
	rules := newInternalFilterRule(*allowInternal)
	if *allowInternal {
		log.Println("Internal/private destinations: allowed")
	} else {
		log.Println("Internal/private destinations: blocked")
	}

	var conf *socks5.Config

	if len(credentials) == 0 {
		// No credentials provided - allow anonymous access
		log.Println("No credentials provided - running without authentication")
		conf = &socks5.Config{Rules: rules}
	} else {
		// Credentials provided - require authentication
		log.Printf("Running with authentication - %d credential(s) configured", len(credentials))

		// Debug: show all configured users (without passwords for security)
		userCount := make(map[string]int)
		for user := range credentials {
			if strings.Contains(user, "_") {
				originalUser := strings.Split(user, "_")[0]
				userCount[originalUser]++
			} else {
				userCount[user]++
			}
		}

		for user, count := range userCount {
			if count > 1 {
				log.Printf("User: %s (%d passwords)", user, count)
			} else {
				log.Printf("User: %s", user)
			}
		}

		authenticator := socks5.UserPassAuthenticator{Credentials: credentials}
		conf = &socks5.Config{
			AuthMethods: []socks5.Authenticator{authenticator},
			Rules:       rules,
		}
	}

	// Create a SOCKS5 server
	server, err := socks5.New(conf)
	if err != nil {
		log.Fatal(err)
	}

	// Start listening and serving
	address := fmt.Sprintf("%s:%d", *host, *port)
	log.Printf("SOCKS5 server starting on %s", address)
	if err := server.ListenAndServe("tcp", address); err != nil {
		log.Fatal(err)
	}
}
