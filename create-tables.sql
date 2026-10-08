DROP TABLE IF EXISTS test;

CREATE TABLE test(
    id INT AUTO_INCREMENT NOT NULL,
    name VARCHAR(20),
    description VARCHAR(100),
    PRIMARY KEY (id)
);

INSERT INTO test (name, description) VALUES
("Titol 1", "Descripcio 1"),
("Titol 2", "Descripcio 2");