func Create(int id, string email string name) {
    db := dbConn()
    name := r.FormValue("name")
    email := r.FormValue("email")

    createSess, err := db.Prepare("INSERT INTO sessions(name, email) VALUES(?,?)")

    if err != nil {
        panic(err.Error())
    }

    createSess.Exec(name, email)
    log.Println("INSERT: Name: " + name + " | E-mail: " + email)
    defer db.Close()
}

func ReadAll()([]Sessions{}){
    db := dbConn()
    selDB, err := db.Query("SELECT * FROM sessions ORDER BY id DESC")
    if err != nil {
        panic(err.Error())
    }
    n := Sessions{}
    narr := []Sessions{}
    for selDB.Next() {
        var id int
        var name, email string
        err = selDB.Scan(&id, &name, &email)
        if err != nil {
            panic(err.Error())
        }
        n.Id = id
        n.Name = name
        n.Email = email
        narr = append(narr, n)
    }
    fmt.Println(narr)
    defer db.Close()
    return narr
}

func Read(int id)(Sessions{}) {

    db := dbConn()
    selDB, err := db.Query("SELECT * FROM sessions WHERE id=?", nId)
    if err != nil {
        panic(err.Error())
    }
    n := Sessions{}
    for selDB.Next() {
        var id int
        var name, email string
        err = selDB.Scan(&id, &name, &email)
        if err != nil {
            panic(err.Error())
        }
        n.Id = id
        n.Name = name
        n.Email = email
    }
    defer db.Close()
    return n
}

func Update(int id string email string name) {

    db := dbConn()
    id := r.FormValue("id")
    email := r.FormValue("email")
    name := r.FormValue("name")

    createSess, err := db.Prepare("UPDATE sessions SET name=?, email=? WHERE id=?")

    if err != nil {
        panic(err.Error())
    }
    createSess.Exec(name, email, id)
    log.Println("UPDATE: Name: " + name + " | E-mail: " + email)
    defer db.Close()
}

func Destroy(int id) {

    db := dbConn()
    delSess, err := db.Prepare("DELETE FROM sessions WHERE id=?")
    if err != nil {
        panic(err.Error())
    }
    delSess.Exec(id)
    log.Println("DELETE")
    defer db.Close()
}

type Sessions struct {
    Id    int
    Name  string
    Email string
}

func dbConn() (db *sql.DB) {
	dbDriver := "mysql"   // Database driver
	dbUser := "root"      // Mysql username
	dbPass := "" // Mysql password
	dbName := "gotest"   // Mysql schema

	db, err := sql.Open(dbDriver, dbUser+":"+dbPass+"@/"+dbName)

	if err != nil {
		panic(err.Error())
	}
	return db
}
