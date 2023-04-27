CREATE SCHEMA IF NOT EXISTS gorm_test;
USE gorm_test;

DROP TABLE IF EXISTS `user`;
CREATE TABLE IF NOT EXISTS `user`
(
    `id`         INT unsigned NOT NULL AUTO_INCREMENT COMMENT 'user ID',
    `name`       VARCHAR(64)  NOT NULL COMMENT '钱包地址',
    `level`      INT unsigned NOT NULL COMMENT '用户等级',
    `status`     int unsigned NOT NULL DEFAULT '0' COMMENT '结算状态 0: 正常  20禁用',
    `created_at` bigint       NOT NULL COMMENT '创建时间 毫秒',
    `updated_at` bigint       NOT NULL COMMENT '更新时间 毫秒',
    PRIMARY KEY (`id`),
    key key_created_at (created_at)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb3 COMMENT ='user';

INSERT INTO gorm_test.user (name, level, status, created_at, updated_at)
VALUES
    ('Boo', 1, 1, 1682597017126, 1682597017126),
    ('Foo', 2, 2, 1682697017126, 1682697017126),
    ('Hoo', 3, 3, 1682797017126, 1682797017126),
    ('Ioo', 4, 4, 1682897017126, 1682897017126),
    ('Joo', 5, 5, 1682997017126, 1682997017126),
    ('Koo', 6, 6, 1683097017126, 1683097017126);
