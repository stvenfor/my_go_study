-- schema 列级对齐（parity-p0-1 审计产物）：把迁移声明收拢到 AutoMigrate 实际跑出来的形态。
--
-- 依据：空库跑完 20260924100000 为止的全部迁移，与 dev 库（四套建表机制的真实结果）逐列 diff。
-- 差异只有四个方向，且全是「迁移声明比实际更严 / 更窄」，反向为 0 处：
--   5  列缺（积分抵扣相关，此前只由内嵌 SQL 建）  -> ADD COLUMN IF NOT EXISTS
--   101 列被声明 NOT NULL、实际业务写 NULL      -> DROP NOT NULL
--   34  列类型偏窄（integer->bigint、double precision->numeric、text/character->varchar）
--   6   列多了 DEFAULT 0（dev 无默认值）        -> DROP DEFAULT
-- 必须排在所有建表迁移之后：它 ALTER 的表包含 20260924100000 建的 IM 表。
-- 幂等：对已对齐的老库全是 no-op。

-- 1) 补缺失列
ALTER TABLE wys_mall_order ADD COLUMN IF NOT EXISTS payment_mode smallint DEFAULT 1 NOT NULL;
ALTER TABLE wys_mall_order ADD COLUMN IF NOT EXISTS total_points bigint DEFAULT 0 NOT NULL;
ALTER TABLE wys_mall_order_item ADD COLUMN IF NOT EXISTS line_points bigint DEFAULT 0 NOT NULL;
ALTER TABLE wys_mall_order_item ADD COLUMN IF NOT EXISTS price_points bigint DEFAULT 0 NOT NULL;
ALTER TABLE wys_mall_sku ADD COLUMN IF NOT EXISTS price_points bigint DEFAULT 0 NOT NULL;

-- 2) 放宽 NOT NULL
ALTER TABLE analytics_records ALTER COLUMN category DROP NOT NULL;
ALTER TABLE analytics_records ALTER COLUMN channel DROP NOT NULL;
ALTER TABLE analytics_records ALTER COLUMN notes DROP NOT NULL;
ALTER TABLE analytics_records ALTER COLUMN owner_name DROP NOT NULL;
ALTER TABLE analytics_records ALTER COLUMN owner_team DROP NOT NULL;
ALTER TABLE analytics_records ALTER COLUMN region DROP NOT NULL;
ALTER TABLE analytics_records ALTER COLUMN source_system DROP NOT NULL;
ALTER TABLE analytics_records ALTER COLUMN status DROP NOT NULL;
ALTER TABLE analytics_records ALTER COLUMN sub_category DROP NOT NULL;
ALTER TABLE analytics_records ALTER COLUMN subtitle DROP NOT NULL;
ALTER TABLE analytics_records ALTER COLUMN tag_primary DROP NOT NULL;
ALTER TABLE analytics_records ALTER COLUMN tag_secondary DROP NOT NULL;
ALTER TABLE users ALTER COLUMN created_at DROP NOT NULL;
ALTER TABLE users ALTER COLUMN updated_at DROP NOT NULL;
ALTER TABLE wys_after_sales_record ALTER COLUMN content DROP NOT NULL;
ALTER TABLE wys_after_sales_record ALTER COLUMN created_at DROP NOT NULL;
ALTER TABLE wys_after_sales_record ALTER COLUMN created_by DROP NOT NULL;
ALTER TABLE wys_after_sales_record ALTER COLUMN customer_name DROP NOT NULL;
ALTER TABLE wys_after_sales_record ALTER COLUMN customer_phone DROP NOT NULL;
ALTER TABLE wys_after_sales_record ALTER COLUMN plate_no DROP NOT NULL;
ALTER TABLE wys_after_sales_record ALTER COLUMN service_date DROP NOT NULL;
ALTER TABLE wys_after_sales_record ALTER COLUMN service_kind DROP NOT NULL;
ALTER TABLE wys_after_sales_record ALTER COLUMN store_id DROP NOT NULL;
ALTER TABLE wys_after_sales_record ALTER COLUMN title DROP NOT NULL;
ALTER TABLE wys_after_sales_record ALTER COLUMN updated_at DROP NOT NULL;
ALTER TABLE wys_deal_invoice ALTER COLUMN created_at DROP NOT NULL;
ALTER TABLE wys_deal_invoice ALTER COLUMN customer_id DROP NOT NULL;
ALTER TABLE wys_deal_invoice ALTER COLUMN customer_name DROP NOT NULL;
ALTER TABLE wys_deal_invoice ALTER COLUMN customer_phone DROP NOT NULL;
ALTER TABLE wys_deal_invoice ALTER COLUMN status DROP NOT NULL;
ALTER TABLE wys_deal_invoice ALTER COLUMN store_id DROP NOT NULL;
ALTER TABLE wys_deal_invoice ALTER COLUMN submitted_at DROP NOT NULL;
ALTER TABLE wys_deal_invoice ALTER COLUMN updated_at DROP NOT NULL;
ALTER TABLE wys_deal_invoice ALTER COLUMN uploader_user_id DROP NOT NULL;
ALTER TABLE wys_im_free_group ALTER COLUMN created_at DROP NOT NULL;
ALTER TABLE wys_im_free_group ALTER COLUMN updated_at DROP NOT NULL;
ALTER TABLE wys_im_free_group_member ALTER COLUMN joined_at DROP NOT NULL;
ALTER TABLE wys_im_friend_request ALTER COLUMN created_at DROP NOT NULL;
ALTER TABLE wys_im_friend_request ALTER COLUMN updated_at DROP NOT NULL;
ALTER TABLE wys_im_friendship ALTER COLUMN created_at DROP NOT NULL;
ALTER TABLE wys_membership_entitlement ALTER COLUMN apple_original_transaction_id DROP NOT NULL;
ALTER TABLE wys_membership_entitlement ALTER COLUMN expires_at DROP NOT NULL;
ALTER TABLE wys_membership_entitlement ALTER COLUMN huawei_purchase_token DROP NOT NULL;
ALTER TABLE wys_membership_entitlement ALTER COLUMN huawei_subscription_id DROP NOT NULL;
ALTER TABLE wys_membership_entitlement ALTER COLUMN source_channel DROP NOT NULL;
ALTER TABLE wys_membership_entitlement ALTER COLUMN updated_at DROP NOT NULL;
ALTER TABLE wys_membership_order ALTER COLUMN amount_fen DROP NOT NULL;
ALTER TABLE wys_membership_order ALTER COLUMN channel DROP NOT NULL;
ALTER TABLE wys_membership_order ALTER COLUMN created_at DROP NOT NULL;
ALTER TABLE wys_membership_order ALTER COLUMN out_trade_no DROP NOT NULL;
ALTER TABLE wys_membership_order ALTER COLUMN plan_id DROP NOT NULL;
ALTER TABLE wys_membership_order ALTER COLUMN status DROP NOT NULL;
ALTER TABLE wys_membership_order ALTER COLUMN tier DROP NOT NULL;
ALTER TABLE wys_membership_order ALTER COLUMN user_id DROP NOT NULL;
ALTER TABLE wys_new_car_follow_file ALTER COLUMN budget_note DROP NOT NULL;
ALTER TABLE wys_new_car_follow_file ALTER COLUMN closed_reason DROP NOT NULL;
ALTER TABLE wys_new_car_follow_file ALTER COLUMN created_at DROP NOT NULL;
ALTER TABLE wys_new_car_follow_file ALTER COLUMN customer_id DROP NOT NULL;
ALTER TABLE wys_new_car_follow_file ALTER COLUMN customer_name DROP NOT NULL;
ALTER TABLE wys_new_car_follow_file ALTER COLUMN customer_phone DROP NOT NULL;
ALTER TABLE wys_new_car_follow_file ALTER COLUMN follow_level DROP NOT NULL;
ALTER TABLE wys_new_car_follow_file ALTER COLUMN owner_user_id DROP NOT NULL;
ALTER TABLE wys_new_car_follow_file ALTER COLUMN source DROP NOT NULL;
ALTER TABLE wys_new_car_follow_file ALTER COLUMN stage DROP NOT NULL;
ALTER TABLE wys_new_car_follow_file ALTER COLUMN store_id DROP NOT NULL;
ALTER TABLE wys_new_car_follow_file ALTER COLUMN updated_at DROP NOT NULL;
ALTER TABLE wys_new_car_follow_file ALTER COLUMN vehicle_interest DROP NOT NULL;
ALTER TABLE wys_new_car_follow_log ALTER COLUMN author_user_id DROP NOT NULL;
ALTER TABLE wys_new_car_follow_log ALTER COLUMN body DROP NOT NULL;
ALTER TABLE wys_new_car_follow_log ALTER COLUMN created_at DROP NOT NULL;
ALTER TABLE wys_new_car_follow_log ALTER COLUMN file_id DROP NOT NULL;
ALTER TABLE wys_new_car_follow_log ALTER COLUMN follow_level DROP NOT NULL;
ALTER TABLE wys_push_devices ALTER COLUMN alias DROP NOT NULL;
ALTER TABLE wys_push_devices ALTER COLUMN created_at DROP NOT NULL;
ALTER TABLE wys_push_devices ALTER COLUMN device_id DROP NOT NULL;
ALTER TABLE wys_push_devices ALTER COLUMN mock DROP NOT NULL;
ALTER TABLE wys_push_devices ALTER COLUMN platform DROP NOT NULL;
ALTER TABLE wys_push_devices ALTER COLUMN registration_id DROP NOT NULL;
ALTER TABLE wys_push_devices ALTER COLUMN updated_at DROP NOT NULL;
ALTER TABLE wys_push_devices ALTER COLUMN user_id DROP NOT NULL;
ALTER TABLE wys_short_video_likes ALTER COLUMN created_at DROP NOT NULL;
ALTER TABLE wys_short_videos ALTER COLUMN created_at DROP NOT NULL;
ALTER TABLE wys_short_videos ALTER COLUMN updated_at DROP NOT NULL;
ALTER TABLE wys_used_car_order ALTER COLUMN amount DROP NOT NULL;
ALTER TABLE wys_used_car_order ALTER COLUMN created_at DROP NOT NULL;
ALTER TABLE wys_used_car_order ALTER COLUMN customer_id DROP NOT NULL;
ALTER TABLE wys_used_car_order ALTER COLUMN customer_name DROP NOT NULL;
ALTER TABLE wys_used_car_order ALTER COLUMN customer_phone DROP NOT NULL;
ALTER TABLE wys_used_car_order ALTER COLUMN kind DROP NOT NULL;
ALTER TABLE wys_used_car_order ALTER COLUMN mileage_km DROP NOT NULL;
ALTER TABLE wys_used_car_order ALTER COLUMN model_year DROP NOT NULL;
ALTER TABLE wys_used_car_order ALTER COLUMN plate_no DROP NOT NULL;
ALTER TABLE wys_used_car_order ALTER COLUMN status DROP NOT NULL;
ALTER TABLE wys_used_car_order ALTER COLUMN store_id DROP NOT NULL;
ALTER TABLE wys_used_car_order ALTER COLUMN submitted_at DROP NOT NULL;
ALTER TABLE wys_used_car_order ALTER COLUMN updated_at DROP NOT NULL;
ALTER TABLE wys_used_car_order ALTER COLUMN uploader_user_id DROP NOT NULL;
ALTER TABLE wys_used_car_order ALTER COLUMN vehicle_model DROP NOT NULL;
ALTER TABLE wys_used_car_order ALTER COLUMN vin DROP NOT NULL;
ALTER TABLE wys_user_store_stats ALTER COLUMN created_at DROP NOT NULL;
ALTER TABLE wys_user_store_stats ALTER COLUMN updated_at DROP NOT NULL;

-- 3) 类型对齐到实际形态
ALTER TABLE analytics_records ALTER COLUMN metric_bounce_rate TYPE numeric;
ALTER TABLE analytics_records ALTER COLUMN metric_cost TYPE numeric;
ALTER TABLE analytics_records ALTER COLUMN metric_revenue TYPE numeric;
ALTER TABLE analytics_records ALTER COLUMN metric_roi TYPE numeric;
ALTER TABLE analytics_records ALTER COLUMN score_quality TYPE numeric;
ALTER TABLE analytics_records ALTER COLUMN score_risk TYPE numeric;
ALTER TABLE users ALTER COLUMN current_store_id TYPE bigint;
ALTER TABLE users ALTER COLUMN failed_login_count TYPE bigint;
ALTER TABLE wys_after_sales_record ALTER COLUMN mileage TYPE bigint;
ALTER TABLE wys_after_sales_record ALTER COLUMN store_id TYPE bigint;
ALTER TABLE wys_deal_invoice ALTER COLUMN store_id TYPE bigint;
ALTER TABLE wys_im_free_group ALTER COLUMN name TYPE character varying;
ALTER TABLE wys_im_free_group ALTER COLUMN owner_user_id TYPE character varying;
ALTER TABLE wys_im_friend_request ALTER COLUMN from_user_id TYPE character varying;
ALTER TABLE wys_im_friend_request ALTER COLUMN status TYPE character varying;
ALTER TABLE wys_im_friend_request ALTER COLUMN to_user_id TYPE character varying;
ALTER TABLE wys_im_store_group ALTER COLUMN group_id TYPE character varying;
ALTER TABLE wys_new_car_follow_file ALTER COLUMN follow_level TYPE character varying;
ALTER TABLE wys_new_car_follow_file ALTER COLUMN store_id TYPE bigint;
ALTER TABLE wys_new_car_follow_log ALTER COLUMN follow_level TYPE character varying;
ALTER TABLE wys_push_devices ALTER COLUMN alias TYPE character varying;
ALTER TABLE wys_push_devices ALTER COLUMN device_id TYPE character varying;
ALTER TABLE wys_push_devices ALTER COLUMN platform TYPE character varying;
ALTER TABLE wys_push_devices ALTER COLUMN registration_id TYPE character varying;
ALTER TABLE wys_push_devices ALTER COLUMN user_id TYPE character varying;
ALTER TABLE wys_short_videos ALTER COLUMN aspect_ratio TYPE numeric;
ALTER TABLE wys_short_videos ALTER COLUMN like_count TYPE bigint;
ALTER TABLE wys_used_car_order ALTER COLUMN mileage_km TYPE bigint;
ALTER TABLE wys_used_car_order ALTER COLUMN model_year TYPE bigint;
ALTER TABLE wys_used_car_order ALTER COLUMN store_id TYPE bigint;
ALTER TABLE wys_user_store_stats ALTER COLUMN days_joined TYPE bigint;
ALTER TABLE wys_user_store_stats ALTER COLUMN employee_count TYPE bigint;
ALTER TABLE wys_user_store_stats ALTER COLUMN store_days TYPE bigint;
ALTER TABLE wys_user_store_stats ALTER COLUMN total_customers TYPE bigint;

-- 4) 去掉实际不存在的默认值
ALTER TABLE analytics_records ALTER COLUMN created_at DROP DEFAULT;
ALTER TABLE analytics_records ALTER COLUMN observed_at DROP DEFAULT;
ALTER TABLE analytics_records ALTER COLUMN published_at DROP DEFAULT;
ALTER TABLE analytics_records ALTER COLUMN updated_at DROP DEFAULT;
ALTER TABLE analytics_records ALTER COLUMN window_end DROP DEFAULT;
ALTER TABLE analytics_records ALTER COLUMN window_start DROP DEFAULT;
