# BlPay Go

BlPay şəxsi maliyyə idarəetmə tətbiqinin Go ilə yenidən qurulmuş versiyasıdır. Responsive interfeys mobil, planşet və desktop ekranlarına uyğunlaşır. Sayt: [blpay.vercel.app](https://blpay.vercel.app/).

## Bölmələr

- Hesab yaratma, giriş və təhlükəsiz sessiya kukisi
- Gəlir və xərc qeydləri, kateqoriyalar və aylıq limitlər
- Maliyyə hədəfləri və hədəf məbləğini artırma/azaltma
- Aylıq xülasə və kateqoriyalara görə hesabatlar
- Admin istifadəçi idarəetməsi
- Font Awesome Free ikonları, tətbiqlə birlikdə host olunur

## Texnologiya

- Go 1.23+
- Supabase PostgreSQL
- Vercel Go Functions
- Responsive HTML/CSS
- Font Awesome Free 7.3.1 (solid icons)

> Supabase layihəsinin daxili bazası PostgreSQL-dir. MySQL protokolu dəstəklənmir; köhnə MySQL sxemi `app/schema.sql`-də Supabase PostgreSQL üçün qurulub.

## Lokal işə salma

1. Supabase-də layihə yaradın və Database → Connect bölməsindən Transaction pooler connection string götürün.
2. Verilənlər bazası sxemini Supabase-ə tətbiq edin:

```bash
npx supabase@latest link --project-ref YOUR_PROJECT_REF
npx supabase@latest migration up --remote
```

   Migration faylı `supabase/migrations/` qovluğundadır. Go tətbiqi də cədvəllər yoxdursa ilk sorğuda idempotent sxemi qura bilir.
3. `.env.example` əsasında `.env` yaradın və `DATABASE_URL` ilə `SESSION_SECRET` dəyərlərini doldurun.
4. `.env` faylını shell-ə yükləyib tətbiqi başladın:

```bash
set -a && . ./.env && set +a
go run ./cmd/server
```

5. `http://localhost:8080` ünvanını açın.

## GitHub və Vercel

GitHub repozitoriyası: [bliveks/blpay-go](https://github.com/bliveks/blpay-go). `main` branch-ə push edilən dəyişikliklər Vercel Production-a avtomatik deploy olunur; digər branch-lər Preview deployment yaradır.

Vercel environment variables-da `DATABASE_URL` və `SESSION_SECRET` saxlanılır. `DATABASE_URL` Supabase Dashboard → Connect → Transaction pooler connection string-i (port `6543`) olmalıdır. Go driver prepared statement-ləri söndürür və SSL tələb edir.

## İlk admin

İlk qeydiyyatdan keçən istifadəçi standart olaraq `user` rolunda yaradılır. İlk istifadəçini admin etmək üçün Supabase SQL Editor-də onun e-poçtuna uyğun sorğunu icra edin:

```sql
UPDATE istifadeciler SET rol = 'admin' WHERE email = 'your@email.com';
```

## Qeydlər

- Cədvəllər tətbiqin ilk sorğusunda `app/schema.sql` əsasında qurulur.
- `.env` repoya commit edilmir.
- Font Awesome ikonları `app/web` daxilində saxlanılır və deployment zamanı tətbiqlə birlikdə yayımlanır.
