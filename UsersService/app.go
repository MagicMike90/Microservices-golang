package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"

	"github.com/codegangsta/negroni"
	raven "github.com/getsentry/raven-go"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	pb "github.com/magicmike90/microservice-news/UsersService/user_data"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// App is the struct with app configuration values
type App struct {
	DB     *sqlx.DB
	Router *mux.Router
	Cache  Cache
}

// Initialize create the DB connection and prepare all the routes
func (a *App) Initialize(cache Cache, db *sqlx.DB) {
	a.Cache = cache
	a.DB = db
	a.Router = mux.NewRouter()
}

func (a *App) initializeRoutes() {
	a.Router.HandleFunc("/users", a.getUsers).Methods("GET")
	a.Router.HandleFunc("/user", a.createUser).Methods("POST")
	a.Router.HandleFunc("/user/{id:[0-9]+}",
		a.getUser).Methods("GET")
	a.Router.HandleFunc("/user/{id:[0-9]+}",
		a.updateUser).Methods("PUT")
	a.Router.HandleFunc("/user/{id:[0-9]+}",
		a.deleteUser).Methods("DELETE")
	a.Router.HandleFunc("/healthcheck", a.healthcheck).Methods("GET")
	a.Router.HandleFunc("/sentryerr", a.sentryerr).Methods("GET")
}

func (a *App) healthcheck(w http.ResponseWriter, r *http.Request) {
	var err error
	c := a.Cache.Pool.Get()
	defer c.Close()

	// Check Cache
	_, err = c.Do("PING")

	// Check DB
	err = a.DB.Ping()

	if err != nil {
		http.Error(w, "CRITICAL", http.StatusInternalServerError)
		return
	}

	// If there is no error within the components of the application, we return a message stating that everything is normal:
	w.Write([]byte("OK"))
	return
}

func (a *App) sentryerr(w http.ResponseWriter, r *http.Request) {
	_, err := os.Open("filename.ext")
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte("OK"))
	return
}

func (a *App) Run(addr string) {
	n := negroni.Classic()
	n.UseHandler(a.Router)
	log.Fatal(http.ListenAndServe(addr, n))
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func (a *App) getUserFromCache(id int) (string, error) {
	if value, err := a.Cache.getValue(id); err == nil && len(value) != 0 {
		return value, err
	}
	return "", errors.New("Not Found")
}

func (a *App) getUserFromDB(id int) (User, error) {
	user := User{ID: id}
	if err := user.get(a.DB); err != nil {
		switch err {
		case sql.ErrNoRows:
			return user, err
		default:
			return user, err
		}
	}
	return user, nil
}

func (a *App) getUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid product ID")
		return
	}

	// searches directly from the cache:
	if value, err := a.getUserFromCache(id); err == nil && len(value) != 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(value))
		return
	}

	//  searches the database if no data is found in the cache:
	user, err := a.getUserFromDB(id)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			respondWithError(w, http.StatusNotFound, "User not found")
		default:
			respondWithError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	response, _ := json.Marshal(user)
	if err := a.Cache.setValue(user.ID, response); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

func (a *App) getUsers(w http.ResponseWriter, r *http.Request) {
	count, _ := strconv.Atoi(r.FormValue("count"))
	start, _ := strconv.Atoi(r.FormValue("start"))

	if count > 10 || count < 1 {
		count = 10
	}
	if start < 0 {
		start = 0
	}

	users, err := list(a.DB, start, count)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, users)
}

func (a *App) createUser(w http.ResponseWriter, r *http.Request) {
	var user User
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&user); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	defer r.Body.Close()

	// get sequence from Postgres
	a.DB.Get(&user.ID, "SELECT nextval('users_id_seq')")

	JSONByte, _ := json.Marshal(user)
	if err := a.Cache.setValue(user.ID, string(JSONByte)); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := a.Cache.enqueueValue(createUsersQueue, user.ID); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, user)
}

func (a *App) updateUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid product ID")
		return
	}

	var user User
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&user); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid resquest payload")
		return
	}
	defer r.Body.Close()
	user.ID = id

	if err := user.update(a.DB); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, user)
}

func (a *App) deleteUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid User ID")
		return
	}

	user := User{ID: id}
	if err := user.delete(a.DB); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"result": "success"})
}

func (a *App) runGRPCServer(portAddr string) {
	lis, err := net.Listen("tcp", portAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterGetUserDataServer(s, &userDataHandler{app: a})
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

type userDataHandler struct {
	app *App
}

func (handler *userDataHandler) composeUser(user User) *pb.UserDataResponse {
	return &pb.UserDataResponse{
		Id:    int32(user.ID),
		Email: user.Email,
		Name:  user.Name,
	}
}

func (handler *userDataHandler) GetUser(ctx context.Context, request *pb.UserDataRequest) (*pb.UserDataResponse, error) {
	var user User
	var err error

	if value, err := handler.app.getUserFromCache(int(request.Id)); err == nil {
		if err = json.Unmarshal([]byte(value), &user); err != nil {
			return nil, err
		}
		return handler.composeUser(user), nil
	}

	if user, err = handler.app.getUserFromDB(int(request.Id)); err == nil {
		return handler.composeUser(user), nil
	}
	return nil, err
}
