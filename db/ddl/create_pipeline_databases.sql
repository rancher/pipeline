CREATE DATABASE IF NOT EXISTS `pipeline` COLLATE = 'utf8_general_ci' CHARACTER SET = 'utf8';
GRANT ALL ON pipeline.* TO 'cattle'@'%' IDENTIFIED BY 'cattle';
GRANT ALL ON pipeline.* TO 'cattle'@'localhost' IDENTIFIED BY 'cattle';