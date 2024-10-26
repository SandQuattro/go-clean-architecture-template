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
	livesPath  = "http://" + host + "/livez"
	attempts   = 20

	// HTTP REST
	basePath = "http://" + host + "/v1"
)

func TestMain(m *testing.M) {
	if v, ok := os.LookupEnv("SKIP_INTEGRATION_TESTS"); ok && v != "" {
		log.Println("Skipping integration tests")
		os.Exit(0)
	}

	code := m.Run()
	os.Exit(code)
}

func TestHealthCheck(t *testing.T) {
	err := healthCheck(attempts)
	if err != nil {
		t.Fatalf("Integration healthCheck test: host %s is not available: %v", host, err)
	}
}

func TestLivesCheck(t *testing.T) {
	err := livesCheck(attempts)
	if err != nil {
		t.Fatalf("Integration livesCheck test: host %s is not available: %v", host, err)
	}
}

func healthCheck(attempts int) error {
	var err error

	for attempts > 0 {
		err = Do(Get(healthPath), Expect().Status().Equal(http.StatusOK))
		if err == nil {
			log.Printf("Integration healthCheck test: %s/readyz is available", host)
			return nil
		}
		log.Printf("Integration tests: url %s is not available, attempts left: %d", healthPath, attempts)
		time.Sleep(time.Second)
		attempts--
	}

	return err
}

func livesCheck(attempts int) error {
	var err error

	for attempts > 0 {
		err = Do(Get(livesPath), Expect().Status().Equal(http.StatusOK))
		if err == nil {
			log.Printf("Integration livesCheck test: %s/livez is available", host)
			return nil
		}
		log.Printf("Integration tests: url %s is not available, attempts left: %d", livesPath, attempts)
		time.Sleep(time.Second)
		attempts--
	}

	return err
}
