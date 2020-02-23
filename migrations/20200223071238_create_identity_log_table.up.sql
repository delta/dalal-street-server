CREATE TABLE IdentityLog (
    `userId` int(11) REFERENCES Users(id),
    `information` varchar(255),
    `updatedAt` timestamp DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
