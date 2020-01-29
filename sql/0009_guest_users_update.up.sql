-- Copyright 2020 Tom Peters
-- 
-- Licensed under the Apache License, Version 2.0 (the "License");
-- you may not use this file except in compliance with the License.
-- You may obtain a copy of the License at
-- 
--    http://www.apache.org/licenses/LICENSE-2.0
-- 
-- Unless required by applicable law or agreed to in writing, software
-- distributed under the License is distributed on an "AS IS" BASIS,
-- WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
-- See the License for the specific language governing permissions and
-- limitations under the License.


BEGIN;

ALTER TABLE guest_users DROP CONSTRAINT guest_users_pkey;
ALTER TABLE guest_users ADD CONSTRAINT guest_users_pkey PRIMARY KEY (store, store_id);

CREATE INDEX guest_users_expires_idx ON guest_users (expires);

COMMIT;
