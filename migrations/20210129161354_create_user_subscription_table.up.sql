CREATE TABLE IF NOT EXISTS UserSubscriptions (
    id int(255) UNSIGNED NOT NULL AUTO_INCREMENT,
    userid int(11) UNSIGNED NOT NULL,
    endpoint varchar(255) NOT NULL,
    p256dh varchar(255) NOT NULL,
    auth varchar(255) NOT NULL,
    PRIMARY KEY (id),
    FOREIGN KEY (userId) REFERENCES Users(id)
)