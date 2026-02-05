package main

import (
	"fmt"
	"log"
	"net/http"

	"golang.org/x/net/webdav"
)

func main() {
	// 1. Configura o servidor WebDAV
	// Ele vai usar a pasta "./data" como raiz da drive
	davHandler := &webdav.Handler{
		FileSystem: webdav.Dir("./data"),
		LockSystem: webdav.NewMemLS(),
		Logger: func(r *http.Request, err error) {
			if err != nil {
				log.Printf("ERRO WebDAV [%s]: %v", r.Method, err)
			} else {
				log.Printf("Ação [%s]: %s", r.Method, r.URL.Path)
			}
		},
	}

	// 2. Define a rota
	// O handler do WebDAV toma conta de tudo (GET, PUT, PROPFIND, MKCOL...)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Dica: O Windows por vezes tenta autenticação.
		// Vamos deixar passar tudo por enquanto (Modo Aberto).
		davHandler.ServeHTTP(w, r)
	})

	// 3. Inicia o Servidor
	fmt.Println("Atlas Drive (WebDAV) a correr na porta :8080")
	fmt.Println("Pasta partilhada: ./data")
	fmt.Println("No Windows: Map Network Drive -> http://localhost:8080")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
