package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/malonso1976/fegan-master-team/internal/store"
	"github.com/malonso1976/fegan-master-team/internal/web"
)

func main() {
	if e := run(); e != nil {
		log.Fatal(e)
	}
}
func run() error {
	config, e := os.UserConfigDir()
	if e != nil {
		return e
	}
	data := flag.String("data", filepath.Join(config, "fegan-master-team", "team.sqlite"), "ruta de la base SQLite")
	port := flag.Int("port", 0, "puerto local (0 = disponible)")
	noBrowser := flag.Bool("no-browser", false, "no abrir navegador")
	flag.Parse()
	if *port < 0 || *port > 65535 {
		return fmt.Errorf("puerto fuera de rango")
	}
	db, e := store.Open(*data)
	if e != nil {
		return e
	}
	defer db.Close()
	listener, e := net.Listen("tcp4", net.JoinHostPort("127.0.0.1", strconv.Itoa(*port)))
	if e != nil {
		return fmt.Errorf("no se pudo abrir el puerto; usa -port 0: %w", e)
	}
	defer listener.Close()
	app := web.New(db, listener.Addr().String())
	defer app.Close()
	var browserClosed <-chan struct{}
	if !*noBrowser {
		browserClosed = app.TrackBrowser()
	}
	server := &http.Server{Handler: app.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 180 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 * 1024}
	stop, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	address := "http://" + listener.Addr().String()
	fmt.Printf("FEGAN Master Team\nAbre %s\nDatos: %s\n", address, *data)
	if *noBrowser {
		fmt.Println("Ctrl+C para salir.")
	} else {
		fmt.Println("Cierra la última pestaña de la aplicación para salir (5 segundos de margen).")
	}
	if !*noBrowser {
		var command string
		var args []string
		switch runtime.GOOS {
		case "windows":
			command = "rundll32"
			args = []string{"url.dll,FileProtocolHandler", address}
		case "darwin":
			command = "open"
			args = []string{address}
		default:
			command = "xdg-open"
			args = []string{address}
		}
		cmd := exec.Command(command, args...)
		if e := cmd.Start(); e != nil {
			log.Print("Abre la dirección local en tu navegador.")
		} else {
			go func() { _ = cmd.Wait() }()
		}
	}
	select {
	case e := <-done:
		if e != http.ErrServerClosed {
			return e
		}
	case <-stop.Done():
	case <-browserClosed:
	}
	ctx, c := context.WithTimeout(context.Background(), 5*time.Second)
	defer c()
	if e := server.Shutdown(ctx); e != nil {
		return server.Close()
	}
	return nil
}
