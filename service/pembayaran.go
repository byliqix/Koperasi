package service

import (
	"database/sql"
	"errors"
	"time"

	"github.com/GemaSatya/ProjekPemweb/config"
	models "github.com/GemaSatya/ProjekPemweb/model"
	"github.com/GemaSatya/ProjekPemweb/repository"
	"github.com/GemaSatya/ProjekPemweb/util"
)

// HitungDenda adalah rumus murni: persen per hari dari sisa tagihan, dibulatkan sekali.
func HitungDenda(sisaTagihan int64, hariTerlambat int, persenPerHari float64) int64 {
	if sisaTagihan <= 0 || hariTerlambat <= 0 || persenPerHari <= 0 {
		return 0
	}
	return util.PersenDari(sisaTagihan*int64(hariTerlambat), persenPerHari)
}

// AlokasiPembayaran membagi uang secara berurutan: denda, lalu jasa, lalu pokok.
func AlokasiPembayaran(nominal, dendaSisa, jasaSisa, pokokSisa int64) (denda, jasa, pokok int64, err error) {
	if nominal <= 0 || nominal > dendaSisa+jasaSisa+pokokSisa {
		return 0, 0, 0, ErrNominalMelebihiTagihan
	}

	denda = nominal
	if denda > dendaSisa {
		denda = dendaSisa
	}
	sisa := nominal - denda

	jasa = sisa
	if jasa > jasaSisa {
		jasa = jasaSisa
	}
	sisa -= jasa

	pokok = sisa
	if pokok > pokokSisa {
		pokok = pokokSisa
	}
	return denda, jasa, pokok, nil
}

// HitungStatusJadwal: jadwal lunas hanya jika pokok dan jasa terbayar penuh.
func HitungStatusJadwal(pokokDibayar, pokok, jasaDibayar, jasa int64) string {
	if pokokDibayar >= pokok && jasaDibayar >= jasa {
		return "Lunas"
	}
	return "Belum"
}

type PembayaranJadwalRepository interface {
	GetByID(db repository.DBTX, id int64) (models.JadwalAngsuran, error)
	GetByIDForUpdate(db repository.DBTX, id int64) (models.JadwalAngsuran, error)
	ListDetailByPinjaman(idPinjaman int64) ([]models.JadwalDetail, error)
	UpdateStatus(db repository.DBTX, id int64, status string) error
	CountUnpaid(db repository.DBTX, idPinjaman int64) (int, error)
}

type PembayaranPinjamanRepository interface {
	GetByIDForUpdate(db repository.DBTX, id int64) (models.Pinjaman, error)
	UpdateStatus(db repository.DBTX, id int64, status string) error
}

type PembayaranDataRepository interface {
	Insert(db repository.DBTX, pembayaran *models.Pembayaran) error
	SumByJadwal(db repository.DBTX, idJadwal int64) (denda, jasa, pokok int64, err error)
	ListByPinjaman(idPinjaman int64) ([]models.Pembayaran, error)
}

type PembayaranPelunasanRepository interface {
	GetByIDPinjaman(db repository.DBTX, idPinjaman int64) (models.Pelunasan, error)
}

type pembayaranTxRunner interface {
	RunInTx(fn func(tx repository.DBTX) error) error
}

type sqlPembayaranTxRunner struct {
	db *sql.DB
}

func (r sqlPembayaranTxRunner) RunInTx(fn func(tx repository.DBTX) error) error {
	return models.RunInTx(r.db, func(tx *sql.Tx) error {
		return fn(tx)
	})
}

type PembayaranService struct {
	db                   *sql.DB
	jadwalRepository     PembayaranJadwalRepository
	pinjamanRepository   PembayaranPinjamanRepository
	pembayaranRepository PembayaranDataRepository
	pelunasanRepository  PembayaranPelunasanRepository
	persenDenda          float64
	txRunner             pembayaranTxRunner
}

func NewPembayaranService(
	db *sql.DB,
	jadwalRepository PembayaranJadwalRepository,
	pinjamanRepository PembayaranPinjamanRepository,
	pembayaranRepository PembayaranDataRepository,
	pelunasanRepository PembayaranPelunasanRepository,
	cfg config.Config,
) *PembayaranService {
	return &PembayaranService{
		db:                   db,
		jadwalRepository:     jadwalRepository,
		pinjamanRepository:   pinjamanRepository,
		pembayaranRepository: pembayaranRepository,
		pelunasanRepository:  pelunasanRepository,
		persenDenda:          cfg.DendaPersenPerHari,
		txRunner:             sqlPembayaranTxRunner{db: db},
	}
}

func (s *PembayaranService) GetJadwal(idPinjaman int64) ([]models.JadwalDetail, error) {
	if idPinjaman <= 0 {
		return nil, ErrInputPinjaman
	}
	jadwal, err := s.jadwalRepository.ListDetailByPinjaman(idPinjaman)
	if err != nil {
		return nil, err
	}
	hariIni := time.Now()
	for i := range jadwal {
		if jadwal[i].Status == "Lunas" {
			continue
		}
		// Denda tidak disimpan sebagai tagihan, jadi dihitung ulang sebagai denda berjalan.
		hari := util.HariTerlambat(jadwal[i].JatuhTempo, hariIni)
		denda := HitungDenda(jadwal[i].SisaPokok+jadwal[i].SisaJasa, hari, s.persenDenda)
		jadwal[i].SisaDenda = max64(0, denda-jadwal[i].DendaTerbayar)
	}
	return jadwal, nil
}

func (s *PembayaranService) HitungTagihan(idJadwal int64, tanggalBayar time.Time) (models.Tagihan, error) {
	if idJadwal <= 0 {
		return models.Tagihan{}, ErrJadwalTidakValid
	}
	if tanggalBayar.IsZero() {
		return models.Tagihan{}, ErrTanggalTidakValid
	}

	jadwal, err := s.jadwalRepository.GetByID(s.db, idJadwal)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Tagihan{}, ErrJadwalTidakDitemukan
		}
		return models.Tagihan{}, err
	}
	return s.hitungTagihan(s.db, jadwal, tanggalBayar)
}

func (s *PembayaranService) BayarAngsuran(idJadwal int64, tanggalBayar time.Time, nominal int64) (models.HasilPembayaran, error) {
	if idJadwal <= 0 {
		return models.HasilPembayaran{}, ErrJadwalTidakValid
	}
	if tanggalBayar.IsZero() {
		return models.HasilPembayaran{}, ErrTanggalTidakValid
	}
	if nominal <= 0 {
		return models.HasilPembayaran{}, ErrNominalTidakValid
	}

	var pembayaran models.Pembayaran
	var statusJadwal string
	err := s.txRunner.RunInTx(func(tx repository.DBTX) error {
		// Urutan kunci: jadwal lebih dulu, baru pinjaman.
		jadwal, err := s.jadwalRepository.GetByIDForUpdate(tx, idJadwal)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrJadwalTidakDitemukan
			}
			return err
		}
		if jadwal.Status == "Lunas" {
			return ErrJadwalLunas
		}

		pinjaman, err := s.pinjamanRepository.GetByIDForUpdate(tx, jadwal.IDPinjaman)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrPinjamanTidakDitemukan
			}
			return err
		}
		if pinjaman.Status != "Aktif" || pinjaman.TanggalCair == nil {
			return ErrStatusPinjaman
		}

		tagihan, err := s.hitungTagihan(tx, jadwal, tanggalBayar)
		if err != nil {
			return err
		}
		denda, jasa, pokok, err := AlokasiPembayaran(nominal, tagihan.DendaTerutang, tagihan.SisaJasa, tagihan.SisaPokok)
		if err != nil {
			return err
		}

		pembayaran = models.Pembayaran{
			IDJadwal:     idJadwal,
			TanggalBayar: tanggalBayar,
			TotalBayar:   nominal,
			DendaDibayar: denda,
			JasaDibayar:  jasa,
			PokokDibayar: pokok,
		}
		if err := s.pembayaranRepository.Insert(tx, &pembayaran); err != nil {
			return err
		}

		// Status akhir dihitung dari total terbayar setelah alokasi, sebelum penutupan pinjaman.
		pokokTerbayar := (jadwal.Pokok - tagihan.SisaPokok) + pokok
		jasaTerbayar := (jadwal.Jasa - tagihan.SisaJasa) + jasa
		statusJadwal = HitungStatusJadwal(pokokTerbayar, jadwal.Pokok, jasaTerbayar, jadwal.Jasa)

		// Perbarui status jadwal dulu agar hitungan jadwal belum lunas sudah mutlak.
		if err := s.UpdateStatusJadwal(tx, idJadwal); err != nil {
			return err
		}
		return s.CekDanTutupPinjaman(tx, jadwal.IDPinjaman)
	})
	if err != nil {
		return models.HasilPembayaran{}, err
	}
	return models.HasilPembayaran{Pembayaran: pembayaran, StatusJadwal: statusJadwal}, nil
}

func (s *PembayaranService) UpdateStatusJadwal(db repository.DBTX, idJadwal int64) error {
	jadwal, err := s.jadwalRepository.GetByID(db, idJadwal)
	if err != nil {
		return err
	}
	_, jasaDibayar, pokokDibayar, err := s.pembayaranRepository.SumByJadwal(db, idJadwal)
	if err != nil {
		return err
	}

	status := HitungStatusJadwal(pokokDibayar, jadwal.Pokok, jasaDibayar, jadwal.Jasa)
	if jadwal.Status == status {
		return nil
	}
	return s.jadwalRepository.UpdateStatus(db, idJadwal, status)
}

func (s *PembayaranService) CekDanTutupPinjaman(db repository.DBTX, idPinjaman int64) error {
	belum, err := s.jadwalRepository.CountUnpaid(db, idPinjaman)
	if err != nil {
		return err
	}
	if belum > 0 {
		return nil
	}
	return s.pinjamanRepository.UpdateStatus(db, idPinjaman, "Lunas")
}

// HitungSisaPinjaman mengurangi sisa jadwal dengan pokok yang sudah ditutup lewat pelunasan.
func (s *PembayaranService) HitungSisaPinjaman(idPinjaman int64) (int64, error) {
	if idPinjaman <= 0 {
		return 0, ErrInputPinjaman
	}
	jadwal, err := s.jadwalRepository.ListDetailByPinjaman(idPinjaman)
	if err != nil {
		return 0, err
	}
	var sisa int64
	for _, row := range jadwal {
		sisa += row.SisaPokok
	}

	pelunasan, err := s.pelunasanRepository.GetByIDPinjaman(s.db, idPinjaman)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	if err == nil {
		sisa -= pelunasan.SisaPokok
	}
	return max64(0, sisa), nil
}

func (s *PembayaranService) RiwayatPembayaranPinjaman(idPinjaman int64) ([]models.Pembayaran, error) {
	if idPinjaman <= 0 {
		return nil, ErrInputPinjaman
	}
	return s.pembayaranRepository.ListByPinjaman(idPinjaman)
}

func (s *PembayaranService) hitungTagihan(db repository.DBTX, jadwal models.JadwalAngsuran, tanggalBayar time.Time) (models.Tagihan, error) {
	dendaDibayar, jasaDibayar, pokokDibayar, err := s.pembayaranRepository.SumByJadwal(db, jadwal.IDJadwal)
	if err != nil {
		return models.Tagihan{}, err
	}

	pokokSisa := max64(0, jadwal.Pokok-pokokDibayar)
	jasaSisa := max64(0, jadwal.Jasa-jasaDibayar)
	hari := util.HariTerlambat(jadwal.JatuhTempo, tanggalBayar)
	denda := HitungDenda(pokokSisa+jasaSisa, hari, s.persenDenda)
	dendaSisa := max64(0, denda-dendaDibayar)

	return models.Tagihan{
		IDJadwal:      jadwal.IDJadwal,
		SisaPokok:     pokokSisa,
		SisaJasa:      jasaSisa,
		DendaTerutang: dendaSisa,
		Total:         dendaSisa + jasaSisa + pokokSisa,
	}, nil
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
