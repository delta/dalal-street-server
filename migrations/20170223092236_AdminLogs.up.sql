CREATE TABLE AdminLogs (
    username varchar(255) NULL,
    msg TEXT NOT NULL,
    createdAt DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(username) references Admins(username)
)
