-- Create the database
-- Echo statement to indicate the script is being executed
SELECT 'Initialization script is being executed...' AS 'ScriptStatus';

CREATE DATABASE IF NOT EXISTS Rmsdb;

-- Switch to the MySQL system database
USE mysql;

-- Create a new user with limited privileges
CREATE USER 'new_gouser'@'192.168.128.4' IDENTIFIED BY ${DB_PASSWORD};

-- Grant privileges to the new user on the created database
GRANT ALL PRIVILEGES ON Recordings.* TO ${DB_USER}@${MYSQL_HOST};

-- Flush privileges to apply changes
FLUSH PRIVILEGES;