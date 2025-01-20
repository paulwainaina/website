module example.com/website_backend

go 1.21.3

require (
	example.com/districts v0.0.0-00010101000000-000000000000
	example.com/groups v0.0.0-00010101000000-000000000000
	example.com/members v0.0.0-00010101000000-000000000000
	example.com/users v0.0.0-00010101000000-000000000000
	github.com/astaxie/beego v1.12.3
	github.com/joho/godotenv v1.5.1
)

require golang.org/x/text v0.21.0 // indirect

require (
	github.com/golang/snappy v0.0.4 // indirect
	github.com/klauspost/compress v1.16.7 // indirect
	github.com/montanaflynn/stats v0.7.1 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78 // indirect
	go.mongodb.org/mongo-driver v1.17.2
	golang.org/x/crypto v0.32.0 // indirect
	golang.org/x/sync v0.10.0 // indirect; indirect1000000-000000000000
)

replace (
	example.com/districts => ./modules/districts
	example.com/groups => ./modules/groups
	example.com/members => ./modules/members
	example.com/users => ./modules/users
)
