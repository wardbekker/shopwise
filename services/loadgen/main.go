package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

var productIDs = []string{"p1", "p2", "p3", "p4", "p5", "p6", "p7", "p8", "p9", "p10"}

func main() {
	frontendURL := getenv("FRONTEND_URL", "http://frontend:8080")
	rps, _ := strconv.ParseFloat(getenv("RPS", "1"), 64)
	users, _ := strconv.Atoi(getenv("USERS", "5"))
	port := getenv("PORT", "8090")
	if users <= 0 {
		users = 1
	}
	if rps <= 0 {
		rps = 1
	}

	// Each user issues ~ (1 browse + ~2 adds + 1 checkout) = ~4 requests per loop.
	// We pace each user's sleep so total population RPS matches target.
	reqsPerLoop := 4.0
	perUserLoopRate := rps / float64(users) / reqsPerLoop
	if perUserLoopRate <= 0 {
		perUserLoopRate = 0.05
	}
	sleepBetweenLoops := time.Duration(float64(time.Second) / perUserLoopRate)

	log.Printf("loadgen starting: frontend=%s users=%d rps=%.2f sleep=%s", frontendURL, users, rps, sleepBetweenLoops)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	srv := &http.Server{Addr: ":" + port, Handler: mux}
	go func() {
		log.Printf("health endpoint on :%s", port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("health listen: %v", err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	for i := 0; i < users; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			runUser(ctx, id, frontendURL, sleepBetweenLoops)
		}(i)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Printf("shutting down")
	cancel()
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer shutCancel()
	_ = srv.Shutdown(shutCtx)
	wg.Wait()
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func runUser(ctx context.Context, id int, frontendURL string, sleep time.Duration) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(id)))
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Printf("user %d: cookiejar: %v", id, err)
		return
	}
	client := &http.Client{Jar: jar, Timeout: 10 * time.Second}

	for {
		if ctx.Err() != nil {
			return
		}

		if err := doGet(ctx, client, frontendURL+"/"); err != nil {
			log.Printf("user %d browse: %v", id, err)
			sleepCtx(ctx, sleep)
			continue
		}

		nItems := 1 + rng.Intn(3) // 1..3 items
		for i := 0; i < nItems; i++ {
			pid := productIDs[rng.Intn(len(productIDs))]
			if err := doAdd(ctx, client, frontendURL+"/cart/add", pid); err != nil {
				log.Printf("user %d add %s: %v", id, pid, err)
			}
		}

		if err := doPost(ctx, client, frontendURL+"/checkout", nil); err != nil {
			log.Printf("user %d checkout: %v", id, err)
		} else {
			log.Printf("user=%s checkout ok", shortUID(jar, frontendURL))
		}

		sleepCtx(ctx, sleep)
	}
}

func sleepCtx(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}

func doGet(ctx context.Context, c *http.Client, u string) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}

func doAdd(ctx context.Context, c *http.Client, u, pid string) error {
	form := url.Values{}
	form.Set("product_id", pid)
	form.Set("quantity", "1")
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, u, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}

func doPost(ctx context.Context, c *http.Client, u string, body any) error {
	_ = body
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, u, nil)
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}

func shortUID(jar *cookiejar.Jar, frontendURL string) string {
	u, err := url.Parse(frontendURL)
	if err != nil {
		return "?"
	}
	for _, c := range jar.Cookies(u) {
		if c.Name == "uid" {
			v := c.Value
			if len(v) > 8 {
				return v[:8]
			}
			return v
		}
	}
	return "?"
}
