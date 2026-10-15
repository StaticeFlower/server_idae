# IdeaSpark API

API ฝั่งหลังบ้านของเว็บแชร์ไอเดีย เขียนด้วย Go (`net/http` มาตรฐาน) + GORM เชื่อมกับ Postgres ของ Supabase

## เริ่มใช้งาน

ต้องมี Go 1.24 ขึ้นไป

```bash
# 1) ดึง dependencies (ครั้งแรก)
go mod tidy

# 2) ตั้งค่า
cp .env.example .env     # แล้วแก้ DATABASE_URL และ JWT_SECRET

# 3) สร้างตารางใน Supabase (ดูหัวข้อถัดไป)

# 4) รัน
go run ./cmd/api
```

ตรวจสอบ: `curl http://localhost:8080/healthz` ควรได้ `{"status":"ok"}`

## เชื่อม Supabase

1. สร้างตาราง: เปิด **Supabase Dashboard → SQL Editor** แล้ววางเนื้อหาไฟล์ `migrations/001_init.sql` รัน 1 ครั้ง
   (หรือจะตั้ง `AUTO_MIGRATE=true` ให้ GORM สร้างให้ก็ได้ แต่ให้เลือกอย่างใดอย่างหนึ่ง และ SQL ไฟล์นี้เปิด RLS ให้ด้วย)
2. เอา connection string: **Dashboard → Connect** แล้วคัดลอกมาใส่ `DATABASE_URL`
   - แนะนำ **Session pooler** (พอร์ต 5432) เพราะใช้ได้กับเครือข่าย IPv4 ทั่วไป
   - ถ้าใช้ **Transaction pooler** (พอร์ต 6543) ก็ใช้ได้ เพราะโค้ดเปิด `PreferSimpleProtocol` ไว้แล้ว
   - รหัสผ่านที่มีอักขระพิเศษต้อง URL-encode
3. สร้าง `JWT_SECRET`: `openssl rand -base64 48`

> **เรื่องความปลอดภัยที่สำคัญ:** Supabase เปิดตารางใน schema `public` ให้เรียกผ่าน REST API ได้ ถ้าไม่เปิด RLS คนที่ได้ anon key (ซึ่งอยู่ในเว็บฝั่ง client ได้) จะอ่านตาราง `users` รวมถึง `password_hash` ได้ ไฟล์ SQL เปิด RLS ไว้ให้แล้ว ถ้าสร้างตารางด้วย `AUTO_MIGRATE` ต้องไปรัน `alter table ... enable row level security` เองทั้งสองตาราง

## Endpoints

| Method | Path | Auth | หมายเหตุ |
| --- | --- | --- | --- |
| GET | `/healthz` | - | เช็กการเชื่อมต่อฐานข้อมูล |
| POST | `/auth/register` | - | `{name, email, password}` → `201 {token, user}` |
| POST | `/auth/login` | - | `{email, password}` → `{token, user}` |
| GET | `/auth/me` | Bearer | → `user` |
| GET | `/ideas` | - | query: `q`, `sort` (`newest` `oldest` `price_asc` `price_desc`), `limit` (1-100, ค่าเริ่มต้น 100), `offset` |
| GET | `/ideas/{id}` | - | → `idea` + `seller` |
| POST | `/ideas` | Bearer | `{title, summary, full_detail, price}` → `201 idea` |

Error ทุกตัวอยู่ในรูป `{"message": "..."}` (ภาษาไทย) ซึ่งหน้าเว็บแสดงผลได้ทันที

ตัวอย่าง:

```bash
curl -X POST localhost:8080/auth/register -H 'Content-Type: application/json' \
  -d '{"name":"นภัส","email":"napat@example.com","password":"password123"}'

TOKEN=...   # เอา token จากผลลัพธ์ข้างบน
curl -X POST localhost:8080/ideas -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"title":"แอปแบ่งบิลกลุ่มเพื่อน","summary":"แชทบอทช่วยหารค่าอาหารแบบเรียลไทม์","full_detail":"รายละเอียดอย่างน้อย 30 ตัวอักษร รายละเอียดอย่างน้อย 30 ตัวอักษร","price":0}'

curl 'localhost:8080/ideas?q=บิล&sort=newest'
```

## ต่อกับหน้าเว็บ (idea-market)

ใน `.env.local` ของหน้าเว็บ:

```
NEXT_PUBLIC_USE_MOCK=false
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080
```

และให้ `CORS_ORIGINS` ของ API ตรงกับ origin ของหน้าเว็บ (ค่าเริ่มต้นคือ `http://localhost:3000`)

## โครงสร้าง

```
cmd/api/            main + graceful shutdown
internal/config/    อ่าน env
internal/database/  เชื่อม Supabase ด้วย GORM, AutoMigrate
internal/models/    User, Idea (ตาม ERD)
internal/auth/      bcrypt + JWT (HS256, อายุ 7 วัน)
internal/request/   request body + validation
internal/middleware/ logging, recover, CORS, ตรวจ Bearer token
internal/handlers/  handlers + routes
migrations/         SQL สำหรับ Supabase
```

## ทดสอบ

```bash
go test ./...
```

มี unit test สำหรับ validation, middleware และ config (ยังไม่มี integration test ที่ต่อฐานข้อมูลจริง)

## ข้อจำกัดที่ควรรู้ (MVP)

- **ยังไม่มี rate limiting** ที่ `/auth/login` และ `/auth/register` ควรเพิ่มก่อนเปิดใช้งานจริง (หรือใช้ของ reverse proxy / CDN)
- **`full_detail` ถูกส่งให้ทุกคนที่เรียก API** เพราะ MVP คือแชร์ไอเดียฟรี เมื่อเริ่มขายจริง ต้องซ่อน `full_detail` ของไอเดียที่มีราคาจากคนที่ยังไม่ได้ซื้อ ทั้งใน `/ideas` และ `/ideas/{id}`
- JWT ไม่มี refresh token / การเพิกถอน token ออกจากระบบทำที่ฝั่ง client เท่านั้น
- หน้าเว็บเก็บ token ใน localStorage ซึ่งพอสำหรับ MVP แต่ถ้าต้องการความปลอดภัยสูงขึ้นควรย้ายเป็น httpOnly cookie
- ยังไม่มี endpoint แก้ไข/ลบไอเดีย และไม่มียืนยันอีเมล
