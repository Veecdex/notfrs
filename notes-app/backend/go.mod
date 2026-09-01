module notes-app

go 1.22

require (
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/lib/pq v1.10.9
	golang.org/x/crypto v0.24.0
)

replace golang.org/x/crypto => github.com/golang/crypto v0.24.0
