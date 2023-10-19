package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)


type GameList struct {
	GameID   int     `json:"gameid"`
	Gamename string  `json:"gamename"`
	Price    float64 `json:"price"`
	ImageURL string  `json:"imageurl"`
}

var gameLists []GameList
var Db *sql.DB
const gamePath = "gamedb"
const BasePath = "/api"



func getGameList(gameid int) (*GameList, error)  {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	row := Db.QueryRowContext(ctx, `SELECT gameid, gamename, price, imageurl FROM steamgame WHERE gameid = ?`, gameid)

	game := &GameList{}
	err := row.Scan(
		&game.GameID,
		&game.Gamename,
		&game.Price,
		&game.ImageURL,
	)

	if err == sql.ErrNoRows {
		return nil, nil  
	}else if err != nil{
		log.Println(err)
		return nil, err
	}
	return game, nil
}

func removeGame(GameID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := Db.ExecContext(ctx, `DELETE FROM steamgame WHERE gameid = ?`, GameID)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	return nil
}

func getGame() ([]GameList, error)  {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	results, err := Db.QueryContext(ctx, `SELECT gameid, gamename, price, imageurl FROM steamgame`)

	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	defer results.Close()
	games := make([]GameList, 0)
	for results.Next(){
		var game GameList
		results.Scan(&game.GameID, &game.Gamename, &game.Price, &game.ImageURL)
		games = append(games, game)
	}
	return games, nil	

}

func insertGame(game GameList) (int, error)  {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result, err := Db.ExecContext(ctx, `INSERT INTO steamgame (gameid, gamename, price, imageurl) VALUES (?, ?, ?, ?)`,
		game.GameID,
		game.Gamename,
		game.Price,
		game.ImageURL)

		if err != nil {
			log.Println(err.Error())
			return 0, err
		}
	insertID, err := result.LastInsertId()
	if err != nil {
		log.Println(err.Error)
		return 0, err
	}
	return int(insertID), nil
}

func handlerGames (w http.ResponseWriter, r *http.Request) {	
	switch r.Method{
	case http.MethodGet:
		gameList, err := getGame()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		j, err := json.Marshal(gameList)
		if err != nil {
			log.Fatal(err)
		}
		_, err = w.Write(j)
		if err != nil{
			log.Fatal(err)
		}
	case http.MethodPost:
		var game GameList
		err := json.NewDecoder(r.Body).Decode(&game)
		if err != nil{
			log.Print(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		GameID, err := insertGame(game)
		if err != nil{
			log.Print(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(fmt.Sprintf(`{"gameid":%d}`, GameID)))
	case http.MethodOptions:
		return
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func handlerGame(w http.ResponseWriter, r *http.Request)  {
	urlPathSegment := strings.Split(r.URL.Path, fmt.Sprintf("%s/", gamePath))
	if len(urlPathSegment[1:]) > 1{
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	gamesID, err := strconv.Atoi(urlPathSegment[len(urlPathSegment)-1])
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	switch r.Method {
	case http.MethodGet:
		game, err := getGameList(gamesID)
		if err != nil{
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if game == nil{
			w.WriteHeader(http.StatusNotFound)
			return
		}
		j, err := json.Marshal(game)
		if err !=nil {
			log.Print(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_, err = w.Write(j)
		if err != nil{
			log.Fatal(err)
		}
	case http.MethodDelete:
		err := removeGame(gamesID)
		if err != nil{
			log.Print(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func corsMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Authorization, X-Control")
		handler.ServeHTTP(w, r)
	})
}

func SetupRoutes(apiBasePath string)  {
	gamesHandler := http.HandlerFunc(handlerGames)
	gameHandler := http.HandlerFunc(handlerGame)	
	http.Handle(fmt.Sprintf("%s/%s", apiBasePath, gamePath), corsMiddleware(gamesHandler))	
	http.Handle(fmt.Sprintf("%s/%s/", apiBasePath, gamePath), corsMiddleware(gameHandler))
}

func SetupDB()  {
	var err error
	Db, err = sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/gamelistdb")

	if err != nil {
        log.Fatal(err)
    }
	fmt.Println(Db)
	Db.SetConnMaxLifetime(time.Minute * 3)
	Db.SetMaxOpenConns(10)
	Db.SetMaxIdleConns(10)
	
}

func main() {
	SetupDB()
	SetupRoutes(BasePath)
	log.Fatal(http.ListenAndServe(":8000", nil))
}