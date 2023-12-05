// main.go

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/lib/pq"
)

const (
	host     = "postgres"
	port     = 5432
	user     = "username"
	password = "password"
	dbname   = "database"
)

var db *sql.DB

func init() {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	var err error
	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
	createSchema(db)
}

func createSchema(db *sql.DB) {
	createTableSQL := `
		CREATE TABLE IF NOT EXISTS your_table (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) UNIQUE
		)
	`
	_, err := db.Exec(createTableSQL)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Schema created successfully")
}

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func main() {
	router := mux.NewRouter()

	// Define routes
	router.HandleFunc("/create", CreateUser).Methods("POST")
	router.HandleFunc("/user/{name}", GetUser).Methods("GET")
	router.HandleFunc("/user/{name}", UpdateUser).Methods("PUT")
	router.HandleFunc("/delete/{name}", DeleteUser).Methods("DELETE")

	log.Fatal(http.ListenAndServe(":8080", router))
}

// create: Accepts POST requests with JSON body to create a new user in the database.
func CreateUser(w http.ResponseWriter, r *http.Request) {
	var user User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_, err = db.Exec("INSERT INTO namestable (name) VALUES ($1)", user.Name)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "unique_violation":
				http.Error(w, "Username already exists", http.StatusBadRequest)
				return
			}
		}
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("User created successfully"))
}

// user/{name}: Accepts GET requests with a name parameter in the URL to
// retrieve the corresponding user.
func GetUser(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(mux.Vars(r)["name"], "/")

	var user User
	err := db.QueryRow("SELECT id, name FROM namestable WHERE name = $1", name).Scan(&user.ID, &user.Name)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	response, err := json.Marshal(user)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}

// user/{name}: Accepts PUT requests with a name parameter in the URL and
// JSON body to update an existing user.
func UpdateUser(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(mux.Vars(r)["name"], "/")
	if !CheckIfNameExists(w, name) {
		return
	}

	var user User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	errCheckName := db.QueryRow("SELECT id, name FROM namestable WHERE name = $1", user.Name).Scan(&user.ID, &user.Name)
	if errCheckName != nil {
		http.Error(w, "Failed to update user - That name already exists", http.StatusInternalServerError)
		return
	}
	_, err = db.Exec("UPDATE namestable SET name = $1 WHERE name = $2", user.Name, name)
	if err != nil {
		http.Error(w, "Failed to update user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("User updated successfully"))
}

// delete/{name}: Accepts DELETE requests to delete a user.
func DeleteUser(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(mux.Vars(r)["name"], "/")
	if !CheckIfNameExists(w, name) {
		return
	}

	var user User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err = db.Exec("DELETE FROM namestable WHERE name = $1", name)
	if err != nil {
		fmt.Println("Error deleting user:", err)
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("User deleted successfully"))
}

func CheckIfNameExists(w http.ResponseWriter, name string) bool {
	var user User
	err := db.QueryRow("SELECT id, name FROM namestable WHERE name = $1", name).Scan(&user.ID, &user.Name)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return false
	}
	return true
}
