package domain

// DevDatabaseClearExcludeServices are left running during clear so Redis/Kafka/MySQL scripts can run.
var DevDatabaseClearExcludeServices = []string{
	"docker-redis",
	"docker-kafka",
	"docker-mysql",
}
