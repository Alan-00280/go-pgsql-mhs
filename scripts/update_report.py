import docx

def update_report(doc_path):
    doc = docx.Document(doc_path)

    # 1. Table 0 (Cover / Metadata)
    t0 = doc.tables[0]
    t0.rows[3].cells[1].text = "07-11-2026"
    t0.rows[4].cells[1].text = "https://github.com/Alan-00280/go-pgsql-mhs.git"

    # 2. Table 1 (Versi Komponen)
    t1 = doc.tables[1]
    t1.rows[1].cells[1].text = "1.26.5"
    t1.rows[2].cells[1].text = "Community 9.0.4 (Lokal)"
    t1.rows[3].cells[1].text = "2.9.2"
    t1.rows[4].cells[1].text = "go.mongodb.org/mongo-driver/v2 v2.9.2"
    t1.rows[5].cells[1].text = "Fiber v2.52.15"

    # 3. Table 2 (Penjelasan Field)
    t2 = doc.tables[2]
    # Row 1: student_id
    t2.rows[1].cells[2].text = "Ya"
    t2.rows[1].cells[3].text = "Merujuk students.id di PostgreSQL"
    # Row 2: headline
    t2.rows[2].cells[2].text = "Ya"
    t2.rows[2].cells[3].text = "Ringkasan profil atau keahlian utama mahasiswa (5-150 karakter)"
    # Row 3: bio
    t2.rows[3].cells[2].text = "Ya"
    t2.rows[3].cells[3].text = "Deskripsi biografi atau latar belakang ringkas mahasiswa (5-500 karakter)"
    # Row 4: projects
    t2.rows[4].cells[2].text = "Tidak"
    t2.rows[4].cells[3].text = "Daftar portofolio proyek mahasiswa (maksimal 20 elemen)"
    # Row 5: projects[]._id
    t2.rows[5].cells[2].text = "Ya"
    t2.rows[5].cells[3].text = "Identifier unik dokumen proyek, dipakai oleh endpoint DELETE"
    # Row 6: certificates
    t2.rows[6].cells[2].text = "Tidak"
    t2.rows[6].cells[3].text = "Daftar sertifikat kompetensi mahasiswa (berisi title, publisher, dan date)"

    # 4. Table 3 (Index yang Dibuat)
    t3 = doc.tables[3]
    t3.rows[1].cells[0].text = "portofolio_student_unique"
    t3.rows[1].cells[1].text = "{ student_id: 1 }"
    t3.rows[1].cells[2].text = "unique: true"
    t3.rows[1].cells[3].text = "Menjamin satu mahasiswa hanya memiliki satu dokumen portofolio serta mempercepat pencarian portofolio pada endpoint GET/PUT /students/:id/portfolio"

    t3.rows[2].cells[0].text = "technology_idx"
    t3.rows[2].cells[1].text = '{ "projects.technologies": 1 }'
    t3.rows[2].cells[2].text = "multikey"
    t3.rows[2].cells[3].text = "Multikey index untuk mempercepat penelusuran array teknologi pada tahapan aggregation pipeline GET /portfolios/stats/technologies"

    # 5. Table 4 (Matriks Pengujian)
    t4 = doc.tables[4]
    t4.rows[1].cells[2].text = "Berhasil mengambil data portofolio mahasiswa"
    t4.rows[1].cells[3].text = "200"

    t4.rows[2].cells[2].text = "Membuat portofolio baru (belum ada data)"
    t4.rows[2].cells[3].text = "201"

    t4.rows[3].cells[2].text = "Mengganti portofolio yang sudah ada"
    t4.rows[3].cells[3].text = "200"

    t4.rows[4].cells[2].text = "Berhasil menambahkan proyek baru ke portofolio"
    t4.rows[4].cells[3].text = "201"

    t4.rows[5].cells[2].text = "Berhasil menghapus proyek dari portofolio"
    t4.rows[5].cells[3].text = "204"

    t4.rows[6].cells[2].text = "Berhasil mengambil statistik 10 teknologi terpopuler"
    t4.rows[6].cells[3].text = "200"

    t4.rows[7].cells[1].text = "GET /students/:id/portfolio (Kasus gagal 1)"
    t4.rows[7].cells[2].text = "Mahasiswa tidak ditemukan di PostgreSQL (id tidak terdaftar)"
    t4.rows[7].cells[3].text = "404"

    t4.rows[8].cells[1].text = "PUT /students/:id/portfolio (Kasus gagal 2)"
    t4.rows[8].cells[2].text = "Body request tidak valid (validasi gagal, headline < 5 karakter)"
    t4.rows[8].cells[3].text = "422"

    t4.rows[9].cells[1].text = "DELETE /students/:id/portfolio/projects/:projectId (Kasus gagal 3)"
    t4.rows[9].cells[2].text = "Format projectId bukan hex ObjectID yang valid"
    t4.rows[9].cells[3].text = "400"

    # 6. Table 5 (Penanganan Data Yatim)
    t5 = doc.tables[5]
    t5.rows[1].cells[0].text = "1. Application-Level Cascade Delete:\nHandler/service penghapusan mahasiswa di PostgreSQL secara eksplisit mengeksekusi perintah penghapusan portofolio terkait di MongoDB (DeleteByStudentID) dalam alur pemanggilan yang sama."
    t5.rows[1].cells[1].text = "Operasi tidak atomik karena ketiadaan transaksi lintas database (cross-database transaction). Jika server padam atau terjadi timeout saat menghapus di MongoDB setelah PostgreSQL sukses di-commit, portofolio tetap menjadi data yatim."

    t5.rows[2].cells[0].text = "2. Asynchronous Event-Driven Cleanup (Transactional Outbox Pattern):\nPenghapusan mahasiswa di PostgreSQL mencatat event ke tabel outbox dalam transaksi yang sama, lalu message broker (seperti RabbitMQ/Kafka) mengonsumsi event tersebut untuk memicu background worker menghapus data di MongoDB dengan mekanisme retry otomatis."
    t5.rows[2].cells[1].text = "Meningkatkan kompleksitas arsitektur sistem (memerlukan message broker dan worker terpisah) serta data portofolio di MongoDB mengalami eventual consistency (terdapat jeda waktu di mana portofolio masih ada sebelum event berhasil diproses worker)."

    # Now let's update paragraphs
    # Paragraph 11: 1. Pendahuluan
    for p in doc.paragraphs:
        txt = p.text.strip()
        if txt.startswith("Jelaskan dalam 2") and "portofolio mahasiswa" in txt:
            p.text = (
                "Fitur yang dibangun adalah sistem pengelolaan portofolio mahasiswa yang mencakup ringkasan profil (headline dan bio), "
                "riwayat proyek (projects), serta sertifikat keahlian (certificates). Data portofolio disimpan di MongoDB karena struktur proyek "
                "dan sertifikat bersifat hierarkis dan dinamis (berupa nested documents di dalam array), sedangkan data entitas mahasiswa tetap "
                "dipertahankan di PostgreSQL untuk menjaga integritas relasional, constraint unik NIM, serta foreign key. Implementasi API ini "
                "dibangun menggunakan Go v1.26.5, framework Fiber v2.52.15, driver resmi go.mongodb.org/mongo-driver/v2 v2.9.2, serta MongoDB "
                "Community Server v9.0.4 (mongosh v2.9.2)."
            )

        # 2.1 projects in mongosh JSON
        elif txt == "projects: [],":
            p.text = (
                "  projects: [\n"
                "    {\n"
                "      _id: ObjectId('6aedfcc36205790b751299eb'),\n"
                "      title: 'Academic Information System REST API',\n"
                "      technologies: [ 'Go', 'Fiber', 'PostgreSQL', 'MongoDB', 'Docker' ],\n"
                "      link: 'https://github.com/Alan-00280/go-pgsql-mhs',\n"
                "      year: 2026,\n"
                "      created_at: ISODate('2026-11-06T16:20:00.000Z')\n"
                "    }\n"
                "  ],"
            )

        # 2.4 Struktur Layer
        elif txt == "[Tuliskan di sini.]":
            # Check context: if it's right under 2.4 Struktur Layer
            p.text = (
                "Arsitektur sistem memisahkan tanggung jawab data ke dalam dua layer yang berbeda. Package model hanya mendefinisikan "
                "struct domain (seperti Portofolio, Project, dan Certificate) yang murni dan bersih dari pustaka/driver pihak ketiga, "
                "menggunakan tipe primitif seperti string untuk ID serta tag json untuk komunikasi API dengan client. Di sisi lain, "
                "package repository mendefinisikan struct dokumen database internal (portofolioDocument, projectDocument, dan certificateDocument) "
                "yang secara eksplisit menggunakan tag bson dan tipe data spesifik driver MongoDB seperti bson.ObjectID untuk pemetaan BSON. "
                "Transformasi antar layer ditangani melalui method pemetaan (seperti toModel()) di layer repository, sehingga isolasi arsitektur "
                "tetap terjaga dan perubahan pada driver database tidak berdampak ke domain model maupun HTTP handler."
            )

    # Let's handle other placeholders carefully by index or unique phrases
    for i, p in enumerate(doc.paragraphs):
        txt = p.text.strip()
        # 3.1
        if i == 135 and txt == "[Penjelasan.]":
            p.text = (
                "Struct portofolioDocument memetakan representasi fisik dokumen BSON di database MongoDB menggunakan bson.ObjectID, "
                "sedangkan method toModel() mengonversinya menjadi entitas domain model.Portofolio dengan merepresentasikan ID sebagai "
                "string heksadesimal agar package model tetap bebas dari dependensi driver MongoDB."
            )
        # 3.2
        elif i == 157 and txt == "[Penjelasan.]":
            p.text = (
                "Method EnsureIndexes mendaftarkan unique index pada student_id untuk menjamin aturan integritas satu mahasiswa satu "
                "dokumen portofolio di level database, serta membuat multikey index pada projects.technologies guna mengoptimalkan "
                "performa query agregasi teknologi."
            )
        # 3.3
        elif i == 223 and "bagaimana kode memastikan hanya satu dokumen" in txt:
            p.text = (
                "Kode memastikan hanya satu dokumen per mahasiswa melalui dua lapisan proteksi: di layer service, method findByStudent "
                "terlebih dahulu memeriksa keberadaan dokumen berdasarkan studentID untuk menentukan operasi Create atau Update, sementara "
                "di level database MongoDB, index unik portofolio_student_unique pada student_id secara mutlak mencegah duplikasi data "
                "meskipun terjadi request konkuren."
            )
        # 3.4
        elif i == 286 and "mengapa batas diletakkan di dalam filter" in txt:
            p.text = (
                "Batas maksimal 20 proyek diletakkan langsung di dalam filter FindOneAndUpdate menggunakan kondisi projects.19: {\"$exists\": false} "
                "agar pengecekan batas dan penambahan proyek ($push) berlangsung secara atomik dalam satu siklus query di database. Jika batas "
                "diperiksa terlebih dahulu di Go lalu baru menulis ke database, akan tercipta celah race condition saat request datang bersamaan "
                "yang berpotensi meloloskan proyek melebihi batas 20."
            )
        # 3.5
        elif i == 322 and txt == "[Penjelasan.]":
            p.text = (
                "Operasi DeleteProject memanfaatkan operator atomik $pull untuk menghapus sub-dokumen proyek dari array projects berdasarkan "
                "projects._id yang cocok, sekaligus memvalidasi hak akses kepemilikan dengan menyertakan filter user_id secara langsung pada query."
            )
        # 3.6
        elif i == 386 and "Penjelasan setiap tahap" in txt:
            p.text = (
                "Aggregation pipeline menggunakan operator $facet untuk menjalankan dua alur analisis secara bersamaan: branch stats "
                "menghitung total dokumen portofolio menggunakan $count, sedangkan branch top_technologies melakukan pemecahan array bertingkat "
                "dengan dua kali $unwind (pada projects kemudian projects.technologies), mengelompokkan frekuensi kemunculan teknologi dengan "
                "$group ($sum: 1), mengurutkan berdasarkan total terbanyak secara menurun dengan $sort, serta membatasi hasilnya menjadi "
                "10 teknologi terpopuler dengan $limit."
            )
        # 3.7
        elif i == 420 and txt == "[Penjelasan.]":
            p.text = (
                "Service memanggil s.students.FindById ke database PostgreSQL untuk memverifikasi keberadaan data mahasiswa sebelum melakukan "
                "operasi portofolio di MongoDB. Hal ini diperlukan karena tidak adanya mekanisme foreign key constraint lintas database relasional "
                "dan NoSQL, sehingga validasi referensial harus ditegakkan secara eksplisit di layer service guna mencegah timbulnya data yatim (orphan portfolio)."
            )

        # 4.1 Curls
        elif i == 426 and "curl [perintah lengkap]" in txt:
            p.text = "curl -X GET http://localhost:3000/api/v1/students/10/portfolio \\\n  -H \"Authorization: Bearer $TOKEN\""
        elif i == 429 and "curl [perintah lengkap]" in txt:
            p.text = (
                "curl -X PUT http://localhost:3000/api/v1/students/10/portfolio \\\n"
                "  -H \"Authorization: Bearer $TOKEN\" \\\n"
                "  -H \"Content-Type: application/json\" \\\n"
                "  -d '{\n"
                "    \"headline\": \"Full-Stack Developer | Building Scalable Web Applications\",\n"
                "    \"bio\": \"A passionate Full-Stack Developer with experience in building modern web applications using Go and PostgreSQL.\",\n"
                "    \"certificates\": [\n"
                "      {\"title\": \"Go Programming Language Fundamentals\", \"publisher\": \"Dicoding Indonesia\", \"date\": \"2025-03-15T00:00:00Z\"}\n"
                "    ]\n"
                "  }'"
            )
        elif i == 432 and "curl [perintah lengkap]" in txt:
            p.text = (
                "curl -X POST http://localhost:3000/api/v1/students/10/portfolio/projects \\\n"
                "  -H \"Authorization: Bearer $TOKEN\" \\\n"
                "  -H \"Content-Type: application/json\" \\\n"
                "  -d '{\n"
                "    \"project\": {\n"
                "      \"title\": \"Academic Information System REST API\",\n"
                "      \"technologies\": [\"Go\", \"Fiber\", \"PostgreSQL\", \"MongoDB\", \"Docker\"],\n"
                "      \"link\": \"https://github.com/Alan-00280/go-pgsql-mhs\",\n"
                "      \"year\": 2026\n"
                "    }\n"
                "  }'"
            )
        elif i == 435 and "curl [perintah lengkap]" in txt:
            p.text = "curl -X DELETE http://localhost:3000/api/v1/students/10/portfolio/projects/6aedfcc36205790b751299eb \\\n  -H \"Authorization: Bearer $TOKEN\""
        elif i == 438 and "curl [perintah lengkap]" in txt:
            p.text = "curl -X GET http://localhost:3000/api/v1/portfolios/stats/technologies \\\n  -H \"Authorization: Bearer $TOKEN\""

        # Failure cases titles & curls
        elif i == 440 and "Kasus gagal 1" in txt:
            p.text = "Kasus gagal 1 — Mahasiswa Tidak Ditemukan di PostgreSQL (404 Not Found)"
        elif i == 441 and "curl [perintah lengkap]" in txt:
            p.text = "curl -X GET http://localhost:3000/api/v1/students/99999/portfolio \\\n  -H \"Authorization: Bearer $TOKEN\""

        elif i == 443 and "Kasus gagal 2" in txt:
            p.text = "Kasus gagal 2 — Validasi Body Request Gagal (422 Unprocessable Entity)"
        elif i == 444 and "curl [perintah lengkap]" in txt:
            p.text = (
                "curl -X PUT http://localhost:3000/api/v1/students/10/portfolio \\\n"
                "  -H \"Authorization: Bearer $TOKEN\" \\\n"
                "  -H \"Content-Type: application/json\" \\\n"
                "  -d '{\"headline\": \"abc\", \"bio\": \"def\", \"certificates\": []}'"
            )

        elif i == 446 and "Kasus gagal 3" in txt:
            p.text = "Kasus gagal 3 — Format Project ID Bukan ObjectID yang Valid (400 Bad Request)"
        elif i == 447 and "curl [perintah lengkap]" in txt:
            p.text = "curl -X DELETE http://localhost:3000/api/v1/students/10/portfolio/projects/invalid-object-id \\\n  -H \"Authorization: Bearer $TOKEN\""

        # 4.2 Bukti Batas 20 Proyek (Skrip Pengiriman Serentak)
        elif i == 452 and "# [tempel skrip pengiriman serentak]" in txt:
            p.text = (
                "# Skrip bash pengiriman 10 request POST secara serentak ke endpoint projects:\n"
                "for i in {1..10}; do\n"
                "  curl -s -o /dev/null -w \"%{http_code}\\n\" -X POST http://localhost:3000/api/v1/students/10/portfolio/projects \\\n"
                "    -H \"Authorization: Bearer $TOKEN\" \\\n"
                "    -H \"Content-Type: application/json\" \\\n"
                "    -d '{\"project\":{\"title\":\"Concurrent Project '$i'\",\"technologies\":[\"Go\"],\"link\":\"https://github.com\",\"year\":2026}}' &\n"
                "done\n"
                "wait"
            )

        # 4.3 NoSQL Injection
        elif i == 464 and "curl [perintah lengkap]" in txt:
            p.text = (
                "curl -X PUT http://localhost:3000/api/v1/students/10/portfolio \\\n"
                "  -H \"Authorization: Bearer $TOKEN\" \\\n"
                "  -H \"Content-Type: application/json\" \\\n"
                "  -d '{\"headline\": {\"$ne\": \"\"}, \"bio\": \"Testing NoSQL Injection\", \"certificates\": []}'"
            )
        elif i == 466 and "Penjelasan singkat:" in txt:
            p.text = (
                "Penjelasan singkat: Permintaan NoSQL injection ditolak karena request body didekode ke struct Go bertipe statis (CreatePortofolioReq). "
                "Karena field headline didefinisikan sebagai string, pengiriman objek operator seperti {\"$ne\": \"\"} menyebabkan parser JSON (c.BodyParser) "
                "gagal dengan error type mismatch (cannot unmarshal object into Go struct field of type string) dan server mengembalikan 400 Bad Request "
                "sebelum payload sempat menyentuh query MongoDB."
            )

        # 4.4 explain executionStats
        elif i == 469 and "db.student_portfolios.find" in txt:
            p.text = "db.student_portofolios.find({ student_id: 10 }).explain(\"executionStats\")"
        elif i == 471 and "winningPlan.inputStage.indexName" in txt:
            p.text = "winningPlan.stage / indexName : EXPRESS_IXSCAN / portofolio_student_unique"
        elif i == 472 and "totalKeysExamined" in txt:
            p.text = "totalKeysExamined                : 1"
        elif i == 473 and "totalDocsExamined" in txt:
            p.text = "totalDocsExamined                : 1"
        elif i == 474 and "nReturned" in txt:
            p.text = "nReturned                        : 1"
        elif i == 475 and "Kesimpulan:" in txt:
            p.text = (
                "Kesimpulan: Index portofolio_student_unique digunakan karena query melakukan filter berdasarkan student_id. "
                "MongoDB langsung melakukan index scan (EXPRESS_IXSCAN) langsung ke node b-tree tanpa perlu melakukan pemindaian menyeluruh "
                "ke seluruh collection (COLLSCAN). Jumlah dokumen yang diperiksa (totalDocsExamined = 1) persis sama dengan jumlah dokumen "
                "yang dikembalikan (nReturned = 1) dan kunci index yang dibaca (totalKeysExamined = 1), membuktikan query dieksekusi secara optimal "
                "tanpa ada dokumen yang dibaca sia-sia."
            )

        # 5. Analisis Singkat
        # 5.1 (P484)
        elif i == 484 and txt == "[Jawaban.]":
            p.text = (
                "Data projects disarangkan (embedded) di dalam dokumen portofolio pada collection student_portofolios karena data proyek selalu diakses "
                "dan ditampilkan bersamaan dengan data profil portofolio mahasiswa (relasi 1:N bounded), jumlah proyek dibatasi maksimal 20 item per portofolio "
                "sehingga ukuran total dokumen tetap sangat aman di bawah batas 16 MB MongoDB, dan pendekatan ini menghilangkan keharusan melakukan operasi JOIN "
                "($lookup) lintas collection yang mahal. Keputusan embedding ini harus dibalik menjadi collection tersendiri (referencing) apabila kebutuhan berubah "
                "menjadi proyek kolaboratif yang dikerjakan bersama oleh beberapa mahasiswa (relasi N:M), proyek membutuhkan fitur mandiri seperti review/rating bintang "
                "per proyek dan komentar publik yang sering diakses terpisah, atau jika batasan 20 proyek dihapuskan sehingga array berpotensi tumbuh tanpa batas "
                "melampaui limit dokumen MongoDB."
            )
        # 5.2 (P487)
        elif i == 487 and txt == "[Jawaban.]":
            p.text = (
                "Bila seorang mahasiswa dihapus dari PostgreSQL, dokumen portofolio mahasiswa tersebut di collection student_portofolios MongoDB akan tetap tersimpan "
                "dan menjadi data yatim (orphan record). Hal ini disebabkan PostgreSQL dan MongoDB merupakan dua sistem database terpisah (polyglot persistence) yang "
                "tidak memiliki mekanisme referential integrity deklaratif seperti FOREIGN KEY ... ON DELETE CASCADE lintas database."
            )
        # 5.3 (P492)
        elif i == 492 and "[Jawaban." in txt:
            p.text = (
                "Bila MongoDB dimatikan saat server aplikasi berjalan, seluruh endpoint yang berinteraksi dengan database MongoDB—yaitu seluruh endpoint portofolio "
                "(/students/:id/portfolio, /students/:id/portfolio/projects*, dan /portfolios/stats/technologies) serta endpoint ulasan kelas (/courses/:id/reviews*, "
                "/reviews/*)—akan gagal dan menghasilkan response 500 Internal Server Error atau 503 Service Unavailable karena koneksi ke driver MongoDB terputus. "
                "Selain itu, endpoint GET /health akan mengembalikan 503 Service Unavailable dengan pesan \"MongoDB tidak dapat dihubungi\" karena health check memantau "
                "ketersediaan kedua database. Sebaliknya, seluruh endpoint inti yang murni mengakses PostgreSQL—seperti CRUD data mahasiswa (/students*), autentikasi "
                "(/auth/*), manajemen pengguna (/users*), dan data perkuliahan (/courses)—tetap berjalan normal dengan response 200 OK. Perilaku ini memang diinginkan "
                "dalam arsitektur polyglot persistence untuk menjamin isolasi kesalahan (fault isolation / graceful degradation), di mana padamnya layanan penunjang seperti "
                "portofolio atau ulasan tidak melumpuhkan fungsionalitas inti operasional akademik di PostgreSQL."
            )

        # 6. Kendala dan Catatan (P495)
        elif i == 495 and txt == "[Tuliskan di sini.]":
            p.text = (
                "Kendala yang dihadapi selama implementasi adalah mencegah race condition saat penambahan proyek secara serentak agar batas 20 proyek tidak terlampaui; "
                "hal ini diatasi dengan meletakkan syarat batas array projects.19: {\"$exists\": false} langsung di dalam filter FindOneAndUpdate bersama operator $push. "
                "Selain itu, terdapat perbedaan penanganan waktu di mana time.Time Go menyimpan presisi hingga nanodetik sedangkan BSON MongoDB hanya hingga milidetik; "
                "hal ini diselesaikan dengan memotong presisi waktu menggunakan .Truncate(time.Millisecond) sebelum disimpan ke MongoDB."
            )

    doc.save(doc_path)
    print("Report template successfully updated!")

if __name__ == "__main__":
    update_report("college/TM10_Template_Laporan_Tugas10_MongoDB.docx")
