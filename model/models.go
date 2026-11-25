package models

import "time"

type Anggota struct {
	IDAnggota     int64     `json:"id_anggota"`
	NoAnggota     string    `json:"no_anggota"`
	Nama          string    `json:"nama"`
	Alamat        string    `json:"alamat"`
	NoHP          string    `json:"no_hp"`
	TanggalDaftar time.Time `json:"tanggal_daftar"`
	Status        string    `json:"status"`
}

type AnggotaInput struct {
	Nama   string `json:"nama"`
	Alamat string `json:"alamat"`
	NoHP   string `json:"no_hp"`
}

type Simpanan struct {
	IDSimpanan int64     `json:"id_simpanan"`
	IDAnggota  int64     `json:"id_anggota"`
	Jenis      string    `json:"jenis"`
	Tanggal    time.Time `json:"tanggal"`
	Nominal    int64     `json:"nominal"`
}

type SimpananInput struct {
	IDAnggota int64     `json:"id_anggota"`
	Jenis     string    `json:"jenis"`
	Tanggal   time.Time `json:"tanggal"`
	Nominal   int64     `json:"nominal"`
}

type PenarikanInput struct {
	IDAnggota int64     `json:"id_anggota"`
	Tanggal   time.Time `json:"tanggal"`
	Nominal   int64     `json:"nominal"`
}

type RingkasanSimpanan struct {
	TotalPokok     int64 `json:"total_pokok"`
	TotalWajib     int64 `json:"total_wajib"`
	TotalSukarela  int64 `json:"total_sukarela"`
	TotalPenarikan int64 `json:"total_penarikan"`
	SaldoSukarela  int64 `json:"saldo_sukarela"`
	GrandTotal     int64 `json:"grand_total"`
}

type Penarikan struct {
	IDPenarikan int64     `json:"id_penarikan"`
	IDAnggota   int64     `json:"id_anggota"`
	Tanggal     time.Time `json:"tanggal"`
	Nominal     int64     `json:"nominal"`
}

type Pinjaman struct {
	IDPinjaman       int64      `json:"id_pinjaman"`
	IDAnggota        int64      `json:"id_anggota"`
	TanggalPengajuan time.Time  `json:"tanggal_pengajuan"`
	TanggalCair      *time.Time `json:"tanggal_cair"`
	JumlahPinjaman   int64      `json:"jumlah_pinjaman"`
	Tenor            int        `json:"tenor"`
	JasaPerBulan     int64      `json:"jasa_per_bulan"`
	Status           string     `json:"status"`
}

type PinjamanInput struct {
	IDAnggota        int64     `json:"id_anggota"`
	TanggalPengajuan time.Time `json:"tanggal_pengajuan"`
	JumlahPinjaman   int64     `json:"jumlah_pinjaman"`
	Tenor            int       `json:"tenor"`
}

type PinjamanDetail struct {
	Pinjaman
	NamaAnggota string `json:"nama_anggota"`
	NoAnggota   string `json:"no_anggota"`
}

type JadwalAngsuran struct {
	IDJadwal   int64     `json:"id_jadwal"`
	IDPinjaman int64     `json:"id_pinjaman"`
	AngsuranKe int       `json:"angsuran_ke"`
	JatuhTempo time.Time `json:"jatuh_tempo"`
	Pokok      int64     `json:"pokok"`
	Jasa       int64     `json:"jasa"`
	Status     string    `json:"status"`
}

type JadwalDetail struct {
	JadwalAngsuran
	DendaTerbayar int64 `json:"denda_terbayar"`
	JasaTerbayar  int64 `json:"jasa_terbayar"`
	PokokTerbayar int64 `json:"pokok_terbayar"`
	SisaDenda     int64 `json:"sisa_denda"`
	SisaJasa      int64 `json:"sisa_jasa"`
	SisaPokok     int64 `json:"sisa_pokok"`
}

type Tagihan struct {
	IDJadwal      int64 `json:"id_jadwal"`
	SisaPokok     int64 `json:"sisa_pokok"`
	SisaJasa      int64 `json:"sisa_jasa"`
	DendaTerutang int64 `json:"denda_terutang"`
	Total         int64 `json:"total"`
}

type Pembayaran struct {
	IDPembayaran int64     `json:"id_pembayaran"`
	IDJadwal     int64     `json:"id_jadwal"`
	TanggalBayar time.Time `json:"tanggal_bayar"`
	TotalBayar   int64     `json:"total_bayar"`
	DendaDibayar int64     `json:"denda_dibayar"`
	JasaDibayar  int64     `json:"jasa_dibayar"`
	PokokDibayar int64     `json:"pokok_dibayar"`
}

// HasilPembayaran menambahkan status jadwal ke response endpoint bayar.
type HasilPembayaran struct {
	Pembayaran
	StatusJadwal string `json:"status_jadwal"`
}

type RincianPelunasan struct {
	IDPinjaman   int64 `json:"id_pinjaman"`
	SisaPokok    int64 `json:"sisa_pokok"`
	Jasa         int64 `json:"jasa"`
	Administrasi int64 `json:"administrasi"`
	Total        int64 `json:"total"`
}

type FilterLaporan struct {
	Dari      time.Time `json:"dari"`
	Sampai    time.Time `json:"sampai"`
	IDAnggota int64     `json:"id_anggota"`
}

type Pelunasan struct {
	IDPelunasan  int64     `json:"id_pelunasan"`
	IDPinjaman   int64     `json:"id_pinjaman"`
	Tanggal      time.Time `json:"tanggal"`
	SisaPokok    int64     `json:"sisa_pokok"`
	Jasa         int64     `json:"jasa"`
	Administrasi int64     `json:"administrasi"`
	Total        int64     `json:"total"`
}

type KartuPinjaman struct {
	Pinjaman   PinjamanDetail `json:"pinjaman"`
	Jadwal     []JadwalDetail `json:"jadwal"`
	Pembayaran []Pembayaran   `json:"pembayaran"`
	Pelunasan  *Pelunasan     `json:"pelunasan,omitempty"`
	SisaPokok  int64          `json:"sisa_pokok"`
}

type BarisSimpanan struct {
	IDAnggota      int64  `json:"id_anggota"`
	NoAnggota      string `json:"no_anggota"`
	Nama           string `json:"nama"`
	TotalPokok     int64  `json:"total_pokok"`
	TotalWajib     int64  `json:"total_wajib"`
	TotalSukarela  int64  `json:"total_sukarela"`
	TotalPenarikan int64  `json:"total_penarikan"`
	SaldoSukarela  int64  `json:"saldo_sukarela"`
}

type BarisPembayaran struct {
	IDPembayaran int64     `json:"id_pembayaran"`
	TanggalBayar time.Time `json:"tanggal_bayar"`
	NoAnggota    string    `json:"no_anggota"`
	Nama         string    `json:"nama"`
	IDPinjaman   int64     `json:"id_pinjaman"`
	IDJadwal     int64     `json:"id_jadwal"`
	AngsuranKe   int       `json:"angsuran_ke"`
	TotalBayar   int64     `json:"total_bayar"`
	DendaDibayar int64     `json:"denda_dibayar"`
	JasaDibayar  int64     `json:"jasa_dibayar"`
	PokokDibayar int64     `json:"pokok_dibayar"`
}

type Tunggakan struct {
	IDAnggota     int64     `json:"id_anggota"`
	NoAnggota     string    `json:"no_anggota"`
	Nama          string    `json:"nama"`
	IDPinjaman    int64     `json:"id_pinjaman"`
	IDJadwal      int64     `json:"id_jadwal"`
	AngsuranKe    int       `json:"angsuran_ke"`
	JatuhTempo    time.Time `json:"jatuh_tempo"`
	HariTerlambat int       `json:"hari_terlambat"`
	SisaPokok     int64     `json:"sisa_pokok"`
	SisaJasa      int64     `json:"sisa_jasa"`
	DendaDibayar  int64     `json:"denda_dibayar"`
	DendaBerjalan int64     `json:"denda_berjalan"`
}

type PendapatanJasa struct {
	Total   int64            `json:"total"`
	Rincian map[string]int64 `json:"rincian"`
}

type PendapatanDenda struct {
	Total   int64            `json:"total"`
	Rincian map[string]int64 `json:"rincian"`
}

type Admin struct {
	IDAdmin      int64  `json:"id_admin"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"`
	Nama         string `json:"nama"`
}
