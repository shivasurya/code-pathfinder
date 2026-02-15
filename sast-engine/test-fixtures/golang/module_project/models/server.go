package models

import (
	"fmt"
	"net/http"
)

type Server struct {
	Host string
	Port int
}

type User struct {
	ID   int
	Name string
}

func NewServer(host string, port int) *Server {
	return &Server{Host: host, Port: port}
}

func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.Host, s.Port)
	return http.ListenAndServe(addr, nil)
}
