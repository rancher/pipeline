#=================================

#=================================
DROP TABLE IF EXISTS `pipelines`;

CREATE TABLE `pipelines` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(50) NOT NULL DEFAULT '',
  `version` int(5) unsigned NOT NULL,
  `status` varchar(50) NOT NULL,
  `workspace_directory` varchar(500) NOT NULL,
  `create_user` varchar(50) NOT NULL DEFAULT '',
  `create_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `update_user` varchar(50) NOT NULL DEFAULT '',
  `update_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

#=================================

#=================================
DROP TABLE IF EXISTS `pipeline_stages`;

CREATE TABLE `pipeline_stages` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `from_pipeline` int(11) unsigned NOT NULL,
  `stage_name` varchar(50) NOT NULL,
  `stage_type` varchar(50) NOT NULL DEFAULT 'Task',
  `job_config_file` text NOT NULL,
  `from_stage` varchar(200) NOT NULL DEFAULT '',
  `status` varchar(50) NOT NULL DEFAULT '',
  `repository` varchar(500) DEFAULT NULL,
  `branch` varchar(500) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `stage_from_pipeline` (`from_pipeline`),
  CONSTRAINT `stage_from_pipeline` FOREIGN KEY (`from_pipeline`) REFERENCES `pipelines` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

#=================================

#=================================
DROP TABLE IF EXISTS `pipelines_activities`;

CREATE TABLE `pipelines_activities` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `from_pipeline` int(11) unsigned NOT NULL,
  `start_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `current_stage_name` varchar(200) DEFAULT '',
  `status` varchar(50) NOT NULL DEFAULT 'triggered',
  PRIMARY KEY (`id`),
  KEY `activity_from_pipeline` (`from_pipeline`),
  CONSTRAINT `activity_from_pipeline` FOREIGN KEY (`from_pipeline`) REFERENCES `pipelines` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

#=================================

#=================================
DROP TABLE IF EXISTS `pipeline_stage_activities`;

CREATE TABLE `pipeline_stage_activities` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `from_activity` int(11) unsigned NOT NULL,
  `from_stage` int(11) unsigned NOT NULL,
  `status` varchar(50) NOT NULL DEFAULT '',
  `final_output` text,
  `duration` int(11) unsigned DEFAULT NULL,
  `start_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `from_acitvity` (`from_activity`),
  KEY `from_stage` (`from_stage`),
  CONSTRAINT `from_acitvity` FOREIGN KEY (`from_activity`) REFERENCES `pipelines_activities` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION,
  CONSTRAINT `from_stage` FOREIGN KEY (`from_stage`) REFERENCES `pipeline_stages` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION
) ENGINE=InnoDB DEFAULT CHARSET=utf8;