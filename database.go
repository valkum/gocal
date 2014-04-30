package main

import (
  "database/sql"
  "log"
  _ "github.com/mattn/go-sqlite3"
  _ "github.com/davecgh/go-spew/spew"
)


type Database struct {
  conn *sql.DB
}

func database() *Database {
  db := new(Database)

  return db
}

func (db *Database) init() {
  conn, err := sql.Open("sqlite3", "./gocal.db")
  db.conn = conn
  if err != nil {
    log.Fatal(err)
  }
  _, err = db.conn.Exec("CREATE TABLE IF NOT EXISTS rooms (id VARCHAR(255) PRIMARY KEY, address VARCHAR(255), cluster VARCHAR(255), building VARCHAR(255), building_no INTEGER, room VARCHAR(255), room_no INTEGER, floor VARCHAR(255))")
  if err != nil {
    log.Fatal(err)
  }
  _, err = db.conn.Exec("CREATE TABLE IF NOT EXISTS settings (key VARCHAR(255) PRIMARY KEY, value BLOB)")
  if err != nil {
    log.Fatal(err)
  }
}

func (db *Database) get_address (r string) (Room, error) {
  stmt, err := db.conn.Prepare("SELECT * FROM rooms WHERE id = ?")
  if err != nil {
    log.Fatal(err)
  }
  defer stmt.Close()
  result := Room{}
  err = stmt.QueryRow(r).Scan(&result.id, &result.address, &result.cluster, &result.building, &result.building_no, &result.room, &result.room_no, &result.floor)
  if err != nil {
    if(err == sql.ErrNoRows) {
      return result, err
    } else {
      log.Fatal(err)
    }
  }
  return result, nil
}

func (db *Database) set_address (room string, address Room) {
  stmt, err := db.conn.Prepare("INSERT OR REPLACE INTO rooms (id, address, cluster, building, building_no, room, room_no, floor) values (?, ?, ?, ?, ?, ?, ?, ?)")
  if err != nil {
    log.Fatal(err)
  }
  defer stmt.Close()
  _, err = stmt.Exec(room, address.address, address.cluster, address.building, address.building_no, address.room, address.room_no, address.floor)
  if err != nil {
    log.Fatal(err)
  }
  return
}

func (db *Database) get_setting (key string) (string, error) {
  byteResult, err := db._get_setting(key)
  if(err != nil) {
    return "", err
  }
  return string(byteResult), nil
}

func (db *Database) get_encrypted_setting (e *Encryption, key string) (string, error) {
  byteResult, err := db._get_setting(key)
  if(err != nil) {
    return "", err
  }
  byteResult = e.decrypt(byteResult)
  return string(byteResult), nil
}

func (db *Database) _get_setting (key string) ([]byte, error) {
  stmt, err := db.conn.Prepare("SELECT value FROM settings WHERE key = ?")
  if err != nil {
    log.Fatal(err)
  }
  defer stmt.Close()
  var byteResult []byte
  err = stmt.QueryRow(key).Scan(&byteResult)
  if err != nil {
    if(err == sql.ErrNoRows) {
      return nil, sql.ErrNoRows
    } else {
      log.Fatal(err)
    }

  }
  return byteResult, nil
}
func (db *Database) set_setting (key string, val string) {
  db._set_setting(key, []byte(val))
}

func (db *Database) set_encrypted_setting (e *Encryption, key string, val string) {
  encVal := e.encrypt([]byte(val))
  db._set_setting(key, encVal)
  return
}

func (db *Database) _set_setting (key string, val []byte) {
  stmt, err := db.conn.Prepare("INSERT OR REPLACE INTO settings (key, value) values (?, ?)")
  if err != nil {
    log.Fatal(err)
  }
  defer stmt.Close()
  byteVal := []byte(val)
  _, err = stmt.Exec(key, byteVal)
  if err != nil {
    log.Fatal(err)
  }
  return
}




