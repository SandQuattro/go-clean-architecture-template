package integration_test

import (
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	. "github.com/Eun/go-hit"
)

var (
	// Attempts connection
	host       = os.Getenv("HOST") + ":" + os.Getenv("PORT")
	healthPath = "http://" + host + "/readyz"
	attempts   = 20

	// HTTP REST
	basePath = "http://" + host + "/v1"
)

func TestMain(m *testing.M) {
	if v, ok := os.LookupEnv("SKIP_INTEGRATION_TESTS"); ok && v != "" {
		log.Println("Skipping integration tests")
		os.Exit(0)
	}

	err := healthCheck(attempts)
	if err != nil {
		log.Fatalf("Integration tests: host %s is not available: %s", host, err)
	}

	log.Printf("Integration tests: host %s is available", host)

	code := m.Run()
	os.Exit(code)
}

func healthCheck(attempts int) error {
	var err error

	for attempts > 0 {
		err = Do(Get(healthPath), Expect().Status().Equal(http.StatusOK))
		if err == nil {
			return nil
		}

		log.Printf("Integration tests: url %s is not available, attempts left: %d", healthPath, attempts)

		time.Sleep(time.Second)

		attempts--
	}

	return err
}
