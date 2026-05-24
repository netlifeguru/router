module github.com/netlifeguru/router

go 1.25.0

require (
	github.com/netlifeguru/logger v0.1.0
	golang.org/x/sys v0.45.0
)

retract [v1.0.0, v1.0.10] // Legacy releases had incorrect module path casing: github.com/NetLifeGuru/router.
