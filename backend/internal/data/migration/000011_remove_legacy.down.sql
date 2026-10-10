CREATE TABLE admin_account(id bigint PRIMARY KEY CHECK(id=1),username varchar(100) NOT NULL,password_hash text NOT NULL);
