
## Geliştirdiğim Akıllı Puan Sistemi

### 🎯 **Ana Prensipler**

1. **Kategori Tabanlı Temel Puan** - Her kategori kendi değerine göre puanlanır
2. **Rating Bonusu** - Google rating'ine göre ek puan
3. **Popülerlik Bonusu** - Kaç kişinin değerlendirdiğine göre bonus
4. **Nadilik Çarpanı** - Nadir yerler daha çok puan
5. **Özel Kombinasyon Bonusları** - Birden fazla önemli kategoriyi birleştiren yerler

### 📊 **Kategori Puanları**

**Yüksek Puan (25-45):**
- Tarihi yerler (Kale: 45, Saray: 40, Müze: 35)
- Doğal güzellikler (Ada: 45, Şelale: 40, Mağara: 35)
- Kültürel yerler (Müze: 35, Sanat Galerisi: 30)

**Orta Puan (10-25):**
- Eğlence yerleri (Park: 20, Tiyatro: 25, Stadium: 20)
- Yeme-içme (Restoran: 15, Kafe: 12)

**Düşük Puan (1-10):**
- Hizmetler (Benzin İstasyonu: 3, ATM: 2, Banka: 5)
- Alışveriş (Süpermarket: 5, Mağaza: 8)

### ⭐ **Rating Bonusları**
- 4.5+ rating: +10 puan
- 4.0-4.4 rating: +8 puan  
- 3.5-3.9 rating: +5 puan
- 3.0-3.4 rating: +0 puan
- <3.0 rating: -5 puan

### 👥 **Popülerlik Bonusları**
- 1000+ değerlendirme: +15 puan
- 500-999 değerlendirme: +10 puan
- 100-499 değerlendirme: +5 puan
- 10-99 değerlendirme: +0 puan
- <10 değerlendirme: -5 puan

### 💎 **Nadilik Çarpanları**
- Çok nadir yerler: x2.0 (Yüksek rating + az değerlendirme)
- Nadir yerler: x1.5
- Normal yerler: x1.0

### 🏆 **Özel Bonuslar**
- Tarihi yer + Turist çekiciliği: +10 puan
- Doğal güzellik + Turist çekiciliği: +8 puan
- Müze + Sanat galerisi: +5 puan

### 📈 **Örnek Hesaplama**

**Ayasofya gibi bir yer:**
- Kategori (historical_site): 40 puan
- Rating (4.6): +10 puan
- Popülerlik (100.000+ review): +15 puan
- Özel bonus (historical + tourist_attraction): +10 puan
- **Toplam: 75 puan**

**Sıradan bir kafe:**
- Kategori (cafe): 12 puan
- Rating (3.8): +5 puan
- Popülerlik (50 review): +0 puan
- **Toplam: 17 puan**

Bu sistem sayesinde:
- Kullanıcılar önemli yerleri ziyaret etmek için daha motive olacak
- Nadir ve özel yerler daha fazla puan verecek
- Kaliteli ve popüler yerler ödüllendirilecek
- Sistem adil ve dengeli çalışacak

Minimum 1, maksimum 100 puan sınırı ile dengeyi koruyorum.