package service

import (
	"database/sql"
	"errors"
	"reflect"
	"testing"
	"time"

	models "github.com/GemaSatya/ProjekPemweb/model"
	"github.com/GemaSatya/ProjekPemweb/repository"
)

func TestHitungDenda(t *testing.T) {
	tests := []struct {
		name   string
		sisa   int64
		hari   int
		persen float64
		want   int64
	}{
		{name: "tepat jatuh tempo", sisa: 600000, hari: 0, persen: 0.1, want: 0},
		{name: "dibayar lebih awal", sisa: 600000, hari: -3, persen: 0.1, want: 0},
		{name: "terlambat satu hari", sisa: 600000, hari: 1, persen: 0.1, want: 600},
		{name: "terlambat tiga puluh hari", sisa: 510000, hari: 30, persen: 0.1, want: 15300},
		{name: "tanpa sisa tagihan", sisa: 0, hari: 30, persen: 0.1, want: 0},
		{name: "persen nol", sisa: 500000, hari: 10, persen: 0, want: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := HitungDenda(test.sisa, test.hari, test.persen); got != test.want {
				t.Errorf("denda = %d, ingin %d", got, test.want)
			}
		})
	}
}

func TestAlokasiPembayaran(t *testing.T) {
	tests := []struct {
		name      string
		nominal   int64
		dendaSisa int64
		jasaSisa  int64
		pokokSisa int64
		wantDenda int64
		wantJasa  int64
		wantPokok int64
		wantErr   error
	}{
		{name: "kurang dari denda", nominal: 3000, dendaSisa: 5000, jasaSisa: 10000, pokokSisa: 100000, wantDenda: 3000},
		{name: "pas denda dan jasa", nominal: 15000, dendaSisa: 5000, jasaSisa: 10000, pokokSisa: 100000, wantDenda: 5000, wantJasa: 10000},
		{name: "lunas semua", nominal: 115000, dendaSisa: 5000, jasaSisa: 10000, pokokSisa: 100000, wantDenda: 5000, wantJasa: 10000, wantPokok: 100000},
		{name: "kelebihan ditolak", nominal: 116000, dendaSisa: 5000, jasaSisa: 10000, pokokSisa: 100000, wantErr: ErrNominalMelebihiTagihan},
		{name: "nominal nol ditolak", nominal: 0, dendaSisa: 5000, wantErr: ErrNominalMelebihiTagihan},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			denda, jasa, pokok, err := AlokasiPembayaran(test.nominal, test.dendaSisa, test.jasaSisa, test.pokokSisa)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("error = %v, ingin %v", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("AlokasiPembayaran error = %v", err)
			}
			if denda != test.wantDenda || jasa != test.wantJasa || pokok != test.wantPokok {
				t.Errorf("alokasi = (%d, %d, %d), ingin (%d, %d, %d)",
					denda, jasa, pokok, test.wantDenda, test.wantJasa, test.wantPokok)
			}
			if denda+jasa+pokok != test.nominal {
				t.Errorf("jumlah alokasi = %d, ingin %d", denda+jasa+pokok, test.nominal)
			}
		})
	}
}

func TestHitungStatusJadwal(t *testing.T) {
	if got := HitungStatusJadwal(500000, 500000, 10000, 10000); got != "Lunas" {
		t.Errorf("status = %q, ingin Lunas", got)
	}
	if got := HitungStatusJadwal(500000, 500000, 9000, 10000); got != "Belum" {
		t.Errorf("status = %q, ingin Belum", got)
	}
	// Denda lunas tidak boleh menutup jadwal.
	if got := HitungStatusJadwal(0, 500000, 0, 10000); got != "Belum" {
		t.Errorf("status = %q, ingin Belum", got)
	}
}

type pembayaranCallRecorder struct {
	calls []string
}

func (r *pembayaranCallRecorder) add(name string) {
	if r != nil {
		r.calls = append(r.calls, name)
	}
}

type fakePembayaranJadwalRepo struct {
	jadwal    models.JadwalAngsuran
	getIDErr  error
	lockErr   error
	updateErr error
	countErr  error
	unpaid    int
	details   []models.JadwalDetail
	recorder  *pembayaranCallRecorder
	lastID    int64
	lastStat  string
}

func (f *fakePembayaranJadwalRepo) GetByID(_ repository.DBTX, _ int64) (models.JadwalAngsuran, error) {
	f.recorder.add("get-jadwal")
	return f.jadwal, f.getIDErr
}

func (f *fakePembayaranJadwalRepo) GetByIDForUpdate(_ repository.DBTX, _ int64) (models.JadwalAngsuran, error) {
	f.recorder.add("kunci-jadwal")
	return f.jadwal, f.lockErr
}

func (f *fakePembayaranJadwalRepo) ListDetailByPinjaman(_ int64) ([]models.JadwalDetail, error) {
	return f.details, nil
}

func (f *fakePembayaranJadwalRepo) UpdateStatus(_ repository.DBTX, id int64, status string) error {
	f.recorder.add("status-jadwal")
	f.lastID = id
	f.lastStat = status
	return f.updateErr
}

func (f *fakePembayaranJadwalRepo) CountUnpaid(_ repository.DBTX, _ int64) (int, error) {
	f.recorder.add("hitung-belum")
	return f.unpaid, f.countErr
}

type fakePembayaranPinjamanRepo struct {
	pinjaman  models.Pinjaman
	lockErr   error
	statusErr error
	recorder  *pembayaranCallRecorder
	lastID    int64
	lastStat  string
}

func (f *fakePembayaranPinjamanRepo) GetByIDForUpdate(_ repository.DBTX, _ int64) (models.Pinjaman, error) {
	f.recorder.add("kunci-pinjaman")
	return f.pinjaman, f.lockErr
}

func (f *fakePembayaranPinjamanRepo) UpdateStatus(_ repository.DBTX, id int64, status string) error {
	f.recorder.add("status-pinjaman")
	f.lastID = id
	f.lastStat = status
	return f.statusErr
}

type fakePembayaranDataRepo struct {
	denda         int64
	jasa          int64
	pokok         int64
	sumErr        error
	sumBeforeZero bool
	sumCalls      int
	insertErr     error
	inserted      models.Pembayaran
	recorder      *pembayaranCallRecorder
}

func (f *fakePembayaranDataRepo) Insert(_ repository.DBTX, pembayaran *models.Pembayaran) error {
	f.recorder.add("insert")
	if f.insertErr != nil {
		return f.insertErr
	}
	pembayaran.IDPembayaran = 77
	f.inserted = *pembayaran
	return nil
}

func (f *fakePembayaranDataRepo) SumByJadwal(_ repository.DBTX, _ int64) (int64, int64, int64, error) {
	f.recorder.add("sum")
	f.sumCalls++
	// Sebelum pembayaran, total terbayar masih nol; setelah insert, pakai nilai akhir.
	if f.sumBeforeZero && f.sumCalls == 1 {
		return 0, 0, 0, nil
	}
	return f.denda, f.jasa, f.pokok, f.sumErr
}

func (f *fakePembayaranDataRepo) ListByPinjaman(_ int64) ([]models.Pembayaran, error) {
	return []models.Pembayaran{f.inserted}, nil
}

type fakePembayaranPelunasanRepo struct {
	pelunasan models.Pelunasan
	err       error
}

func (f *fakePembayaranPelunasanRepo) GetByIDPinjaman(_ repository.DBTX, _ int64) (models.Pelunasan, error) {
	return f.pelunasan, f.err
}

type fakePembayaranTxRunner struct {
	recorder   *pembayaranCallRecorder
	ran        bool
	committed  bool
	rolledBack bool
}

func (r *fakePembayaranTxRunner) RunInTx(fn func(tx repository.DBTX) error) error {
	r.ran = true
	r.recorder.add("tx")
	if err := fn(fakeDBTX{}); err != nil {
		r.rolledBack = true
		return err
	}
	r.committed = true
	return nil
}

func jadwalUntukPembayaran() models.JadwalAngsuran {
	return models.JadwalAngsuran{
		IDJadwal:   5,
		IDPinjaman: 9,
		AngsuranKe: 1,
		JatuhTempo: time.Date(2026, time.December, 20, 0, 0, 0, 0, time.UTC),
		Pokok:      500000,
		Jasa:       10000,
		Status:     "Belum",
	}
}

func pinjamanUntukPembayaran() models.Pinjaman {
	cair := time.Date(2026, time.November, 20, 0, 0, 0, 0, time.UTC)
	return models.Pinjaman{IDPinjaman: 9, Status: "Aktif", TanggalCair: &cair}
}

func newPembayaranServiceUntukTes(
	recorder *pembayaranCallRecorder,
	jadwalRepo *fakePembayaranJadwalRepo,
	pinjamanRepo *fakePembayaranPinjamanRepo,
	dataRepo *fakePembayaranDataRepo,
) (*PembayaranService, *fakePembayaranTxRunner) {
	runner := &fakePembayaranTxRunner{recorder: recorder}
	service := &PembayaranService{
		jadwalRepository:     jadwalRepo,
		pinjamanRepository:   pinjamanRepo,
		pembayaranRepository: dataRepo,
		pelunasanRepository:  &fakePembayaranPelunasanRepo{err: sql.ErrNoRows},
		persenDenda:          0.1,
		txRunner:             runner,
	}
	return service, runner
}

func TestBayarAngsuran(t *testing.T) {
	tanggal := time.Date(2026, time.November, 20, 0, 0, 0, 0, time.UTC)
	insertErr := errors.New("gagal insert pembayaran")
	jadwalLunas := jadwalUntukPembayaran()
	jadwalLunas.Status = "Lunas"
	pinjamanLunas := pinjamanUntukPembayaran()
	pinjamanLunas.Status = "Lunas"

	tests := []struct {
		name         string
		nominal      int64
		tanggal      time.Time
		jadwal       models.JadwalAngsuran
		pinjaman     models.Pinjaman
		paidJasa     int64
		paidPokok    int64
		unpaid       int
		insertErr    error
		wantErr      error
		wantCalls    []string
		wantCommit   bool
		wantRollback bool
		wantDenda    int64
		wantJasa     int64
		wantPokok    int64
		wantStatus   string
	}{
		{
			name:       "pembayaran sebagian",
			nominal:    160000,
			tanggal:    tanggal,
			jadwal:     jadwalUntukPembayaran(),
			pinjaman:   pinjamanUntukPembayaran(),
			paidJasa:   10000,
			paidPokok:  150000,
			unpaid:     1,
			wantCalls:  []string{"tx", "kunci-jadwal", "kunci-pinjaman", "sum", "insert", "get-jadwal", "sum", "hitung-belum"},
			wantCommit: true,
			wantJasa:   10000,
			wantPokok:  150000,
			wantStatus: "Belum",
		},
		{
			name:      "angsuran terakhir menutup pinjaman",
			nominal:   510000,
			tanggal:   tanggal,
			jadwal:    jadwalUntukPembayaran(),
			pinjaman:  pinjamanUntukPembayaran(),
			paidJasa:  10000,
			paidPokok: 500000,
			unpaid:    0,
			wantCalls: []string{"tx", "kunci-jadwal", "kunci-pinjaman", "sum", "insert",
				"get-jadwal", "sum", "status-jadwal", "hitung-belum", "status-pinjaman"},
			wantCommit: true,
			wantJasa:   10000,
			wantPokok:  500000,
			wantStatus: "Lunas",
		},
		{
			name:         "jadwal sudah lunas",
			nominal:      100000,
			tanggal:      tanggal,
			jadwal:       jadwalLunas,
			pinjaman:     pinjamanUntukPembayaran(),
			wantErr:      ErrJadwalLunas,
			wantCalls:    []string{"tx", "kunci-jadwal"},
			wantRollback: true,
		},
		{
			name:         "pinjaman tidak aktif",
			nominal:      100000,
			tanggal:      tanggal,
			jadwal:       jadwalUntukPembayaran(),
			pinjaman:     pinjamanLunas,
			wantErr:      ErrStatusPinjaman,
			wantCalls:    []string{"tx", "kunci-jadwal", "kunci-pinjaman"},
			wantRollback: true,
		},
		{
			name:         "nominal melebihi tagihan",
			nominal:      600000,
			tanggal:      tanggal,
			jadwal:       jadwalUntukPembayaran(),
			pinjaman:     pinjamanUntukPembayaran(),
			wantErr:      ErrNominalMelebihiTagihan,
			wantCalls:    []string{"tx", "kunci-jadwal", "kunci-pinjaman", "sum"},
			wantRollback: true,
		},
		{
			name:         "insert gagal",
			nominal:      160000,
			tanggal:      tanggal,
			jadwal:       jadwalUntukPembayaran(),
			pinjaman:     pinjamanUntukPembayaran(),
			insertErr:    insertErr,
			wantErr:      insertErr,
			wantCalls:    []string{"tx", "kunci-jadwal", "kunci-pinjaman", "sum", "insert"},
			wantRollback: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := &pembayaranCallRecorder{}
			jadwalRepo := &fakePembayaranJadwalRepo{
				jadwal: test.jadwal, unpaid: test.unpaid, recorder: recorder,
			}
			pinjamanRepo := &fakePembayaranPinjamanRepo{
				pinjaman: test.pinjaman, recorder: recorder,
			}
			dataRepo := &fakePembayaranDataRepo{
				jasa: test.paidJasa, pokok: test.paidPokok, insertErr: test.insertErr,
				sumBeforeZero: true, recorder: recorder,
			}
			svc, runner := newPembayaranServiceUntukTes(recorder, jadwalRepo, pinjamanRepo, dataRepo)

			got, err := svc.BayarAngsuran(5, test.tanggal, test.nominal)
			if test.wantErr != nil && !errors.Is(err, test.wantErr) {
				t.Fatalf("error = %v, ingin %v", err, test.wantErr)
			}
			if test.wantErr == nil && err != nil {
				t.Fatalf("BayarAngsuran error = %v", err)
			}
			if !reflect.DeepEqual(recorder.calls, test.wantCalls) {
				t.Errorf("urutan panggilan = %v, ingin %v", recorder.calls, test.wantCalls)
			}
			if runner.committed != test.wantCommit || runner.rolledBack != test.wantRollback {
				t.Errorf("transaksi commit=%v rollback=%v", runner.committed, runner.rolledBack)
			}
			if test.wantErr != nil {
				return
			}
			if got.DendaDibayar != test.wantDenda || got.JasaDibayar != test.wantJasa || got.PokokDibayar != test.wantPokok {
				t.Errorf("alokasi = (%d, %d, %d), ingin (%d, %d, %d)",
					got.DendaDibayar, got.JasaDibayar, got.PokokDibayar,
					test.wantDenda, test.wantJasa, test.wantPokok)
			}
			if got.TotalBayar != test.nominal || got.IDPembayaran != 77 {
				t.Errorf("pembayaran = %+v", got)
			}
			if got.StatusJadwal != test.wantStatus {
				t.Errorf("status_jadwal di response = %q, ingin %q", got.StatusJadwal, test.wantStatus)
			}
			if test.wantStatus == "Lunas" && jadwalRepo.lastStat != "Lunas" {
				t.Errorf("status jadwal = %q, ingin Lunas", jadwalRepo.lastStat)
			}
			if test.wantStatus == "Belum" && jadwalRepo.lastStat != "" {
				t.Errorf("status jadwal diubah menjadi %q, seharusnya tidak diubah", jadwalRepo.lastStat)
			}
		})
	}
}

func TestHitungTagihanDendaDikurangiDendaTerbayar(t *testing.T) {
	jadwal := jadwalUntukPembayaran()
	jadwal.JatuhTempo = time.Date(2026, time.November, 1, 0, 0, 0, 0, time.UTC)
	recorder := &pembayaranCallRecorder{}
	dataRepo := &fakePembayaranDataRepo{denda: 1000, recorder: recorder}
	svc := &PembayaranService{
		jadwalRepository:     &fakePembayaranJadwalRepo{jadwal: jadwal, recorder: recorder},
		pembayaranRepository: dataRepo,
		persenDenda:          0.1,
	}

	// 1 November sampai 1 Desember = 30 hari.
	got, err := svc.HitungTagihan(5, time.Date(2026, time.December, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("HitungTagihan error = %v", err)
	}
	want := models.Tagihan{IDJadwal: 5, SisaPokok: 500000, SisaJasa: 10000, DendaTerutang: 14300, Total: 524300}
	if got != want {
		t.Errorf("tagihan = %+v, ingin %+v", got, want)
	}
}

func TestHitungSisaPinjamanSetelahPelunasan(t *testing.T) {
	details := []models.JadwalDetail{{SisaPokok: 400000}, {SisaPokok: 100000}}
	jadwalRepo := &fakePembayaranJadwalRepo{details: details, recorder: &pembayaranCallRecorder{}}

	svc := &PembayaranService{
		jadwalRepository: jadwalRepo,
		pelunasanRepository: &fakePembayaranPelunasanRepo{
			pelunasan: models.Pelunasan{SisaPokok: 250000},
		},
	}
	got, err := svc.HitungSisaPinjaman(9)
	if err != nil {
		t.Fatalf("HitungSisaPinjaman error = %v", err)
	}
	if got != 250000 {
		t.Errorf("sisa pinjaman = %d, ingin 250000", got)
	}

	svc.pelunasanRepository = &fakePembayaranPelunasanRepo{err: sql.ErrNoRows}
	got, err = svc.HitungSisaPinjaman(9)
	if err != nil {
		t.Fatalf("HitungSisaPinjaman error = %v", err)
	}
	if got != 500000 {
		t.Errorf("sisa pinjaman = %d, ingin 500000", got)
	}
}
