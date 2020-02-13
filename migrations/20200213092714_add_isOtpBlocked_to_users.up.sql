ALTER TABLE `Users` ADD `isOtpBlocked` BOOL NOT NULL;
ALTER TABLE `Users` ADD `otpRequestCount` int(11) NOT NULL DEFAULT 0;
