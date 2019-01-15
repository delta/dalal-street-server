ALTER TABLE MortgageDetails 
ADD COLUMN mortgagePrice int(11),
DROP PRIMARY KEY, ADD PRIMARY KEY(id, userId, stockId, mortgagePrice);
