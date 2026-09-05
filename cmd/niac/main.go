// Package main provides the NIAC command-line interface for network device simulation.
package main

func main() {
	info := readVersionInfo()
	services := new(serviceOptions)

	executeRootCommand(newRootCommand(info, services, commandBuilders(info)))
}
