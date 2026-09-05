# Sereality Panel Roadmap

แผนพัฒนาหลักของ Sereality Panel:

1. ระยะที่ 1: เปลี่ยน Branding + ทำระบบ Backup/Rollback + แก้ Security สำคัญ
2. ระยะที่ 2: Dashboard ใหม่ + ระบบ Package/User management
3. ระยะที่ 3: Multi-node และระบบ API ที่เป็นมาตรฐาน
4. ระยะที่ 4: ระบบสมาชิก/Subscription/การขาย ถ้าต้องการทำเชิงธุรกิจ

## Current focus

ระยะที่ 2–4 — Dashboard reference UI, Package/User foundation, Multi-node foundation และ Subscription lifecycle foundation

## สถานะล่าสุด

สถานะรอบนี้: implementation foundation ของระยะที่ 2–4 เสร็จและผ่าน candidate-wide verification แล้ว; ยังไม่ได้ติดตั้งหรือ deploy ลง production

- Dashboard ใช้ธีม Sereality ชมพู/ขาว/ดำ และดึงข้อมูลจริงจาก API เดิม
- Package management backend foundation พร้อม validation และ CRUD API
- Packages และ Nodes UI รองรับ create/edit/delete ผ่าน endpoint จริง พร้อม confirmation ก่อนลบ
- User management พร้อมรายการผู้ใช้, assign package และเปลี่ยนสถานะ โดย DTO ไม่เปิดเผย credential
- Multi-node model/service foundation พร้อมบังคับ HTTPS, hash token และ plan-only snapshot sync ที่เรียงผลลัพธ์แน่นอน
- API v2 มี health, package CRUD, user list/status/package assignment, node CRUD และ subscription list/create/status แบบ response envelope
- Subscription lifecycle พร้อมรายการ, transition rules และ expiration sweep ทุกนาที; หน้า UI ยังไม่ทำ payment/activate อัตโนมัติ
- Login มี bounded sliding-window rate limit และ default admin password ใหม่เก็บเป็น bcrypt
- Login ไม่ยอมรับ bcrypt hash เป็นรหัสผ่าน, migrate legacy password แบบ compare-and-swap และ session ตรวจสถานะผู้ใช้จากฐานข้อมูลอีกครั้ง
- SQLite เปิด foreign-key enforcement, backup/rollback ใช้ snapshot ที่สอดคล้องกับ WAL และเปลี่ยนไฟล์บน Windows ด้วย MoveFileEx แบบ replace/write-through
- Upload DB จำกัด request body ก่อน multipart parsing เพื่อไม่ให้ไฟล์ใหญ่ spool ลง disk ก่อนถูกปฏิเสธ
- HTTP response มี baseline security headers (`nosniff`, frame deny, referrer และ permissions policy)
- Legacy JSON error responses ไม่ส่ง internal error/path/credential detail กลับไปยัง client แต่ยัง log รายละเอียดไว้ฝั่ง server
- Settings user-secret response ไม่ส่ง password hash และการเปลี่ยนรหัสผ่านตรวจ hash จาก DB โดยตรงแม้ session cookie จะเป็น safe projection
- Frontend `HttpUtil` รองรับ GET/POST/PUT/DELETE เพื่อให้หน้า admin เรียก CRUD API ได้ครบ
- Subscription UI/API v2 สามารถสร้างรายการ `pending` จาก user/package จริง และ backend ตรวจ reference ก่อนบันทึก
- Backup snapshot ใช้ SQLite `VACUUM INTO` เพื่อรวม committed WAL data และเปลี่ยนไฟล์ปลายทางแบบ atomic
- DB import จำกัดขนาดไว้ที่ 512 MiB และตัดการเขียนทันทีเมื่อเกินขนาด
- ระบบชำระเงินจริงและการ deploy ยังไม่ทำจนกว่าจะกำหนด provider/กติกาธุรกิจและมีการอนุมัติเพิ่มเติม

## ข้อจำกัดที่ตรวจพบ

- Multi-node ตอนนี้สร้าง sync plan แบบ `dryRun: true` เท่านั้น และต้องได้รับ snapshot ที่มี `complete: true` ทั้งสองฝั่งก่อนวางแผน จึงไม่ตีความข้อมูลที่ดึงมาไม่ครบเป็นคำสั่งลบ; request ที่ขอ apply ถูกปฏิเสธด้วย 501 จนกว่าจะมี transport ที่อนุมัติ
- Full test ผ่านแล้วด้วย `CGO_ENABLED=1` และ LLVM/LLD toolchain ที่มีอยู่ใน Visual Studio Build Tools (`go test -count=1 -p 1 ./...`)
