package repositories

import (
	"context"
    "github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

func ConnectToDB(url string) (*pgx.Conn, error){
	godotenv.Load(".env.example")
	DBConnection, err := pgx.Connect(context.Background(), url)
    if err != nil {
        return nil, err
    }

    return DBConnection, nil
}

func DisconnectFromDB(DBConnection *pgx.Conn){
	DBConnection.Close(context.Background())
}