package db

import "database/sql"

type ApplicationModel struct {
	ID 					string
	Path 				string
	Strategy 			string
	Type 				string
	DockerfilePath 		string 
	BuildCommand		string
	RunCommand			string
	ApplicationPort		int
	Port				int
}

func InitializeDB() (*sql.DB, error) {

	db, err := sql.Open("sqlite3", "/infracon.db")
	if err != nil {
		return nil, err
	}

	sqlStmt := `
		CREATE TABLE IF NOT EXISTS apps (
			id TEXT NOT NULL PRIMARY KEY AUTOINCREMENT,
			path TEXT NOT NULL,
			strategy TEXT,
			type TEXT,
			dockerfile_path TEXT,
			build_command TEXT,
			run_command TEXT,
			application_port int,
			port int
		);
    `

	_, err = db.Exec(sqlStmt)
	if err != nil {
		return nil, err
	}

	return db, err
}

func GetApplicationFromDB(db *sql.DB, applicationId string) (*ApplicationModel, error){
	rows, err := db.Query(`SELECT FROM apps WHERE id = ?`, applicationId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next()

}
