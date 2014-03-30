package mp3

import "io"

type MP3 struct {
	b *bitReader

	syncword           uint16
	ID                 byte
	layer              Layer
	protection_bit     byte
	bitrate_index      byte
	sampling_frequency byte
	padding_bit        byte
	private_bit        byte
	mode               Mode
	mode_extension     byte
	copyright          byte
	original_home      byte
	emphasis           Emphasis
}

func New(r io.Reader) *MP3 {
	b := newBitReader(r)
	return &MP3{
		b: &b,
	}
}

func (m *MP3) Sequence() (audio []float32, err error) {
	for {
		m.frame()
		break
	}
	return
}

func (m *MP3) frame() {
	m.header()
	m.error_check()
	m.audio_data()
}

func (m *MP3) header() {
	m.syncword = uint16(m.b.ReadBits(12))
	m.ID = byte(m.b.ReadBits(1))
	m.layer = Layer(m.b.ReadBits(2))
	m.protection_bit = byte(m.b.ReadBits(1))
	m.bitrate_index = byte(m.b.ReadBits(4))
	m.sampling_frequency = byte(m.b.ReadBits(2))
	m.padding_bit = byte(m.b.ReadBits(1))
	m.private_bit = byte(m.b.ReadBits(1))
	m.mode = Mode(m.b.ReadBits(2))
	m.mode_extension = byte(m.b.ReadBits(2))
	m.copyright = byte(m.b.ReadBits(1))
	m.original_home = byte(m.b.ReadBits(1))
	m.emphasis = Emphasis(m.b.ReadBits(2))
}

func (m *MP3) error_check() {
	if m.protection_bit == 0 {
		m.b.ReadBits(16)
	}
}

func (m *MP3) audio_data() {
	if m.mode == ModeSingle {
		main_data_end := uint16(m.b.ReadBits(9))
		m.b.ReadBits(5) // private_bits
		scfsi := make([]byte, cblimit)
		var part2_3_length [2]uint16
		var big_values [2]uint16
		var global_gain [2]uint16
		var scalefac_compress [2]byte
		var blocksplit_flag [2]byte
		var block_type [2]byte
		var switch_point [2]byte
		var table_select [3][2]byte
		var subblock_gain [3][2]uint8
		var region_address1, region_address2 [2]byte
		var preflag, scalefac_scale, count1table_select [2]byte
		var scalefac [][2]uint8
		var scalefacw [][3][2]uint8
		for scfsi_band := 0; scfsi_band < 4; scfsi_band++ {
			scfsi[scfsi_band] = byte(m.b.ReadBits(1))
		}
		for gr := 0; gr < 2; gr++ {
			part2_3_length[gr] = uint16(m.b.ReadBits(12))
			big_values[gr] = uint16(m.b.ReadBits(9))
			global_gain[gr] = uint16(m.b.ReadBits(8))
			scalefac_compress[gr] = byte(m.b.ReadBits(4))
			blocksplit_flag[gr] = byte(m.b.ReadBits(1))
			if blocksplit_flag[gr] != 0 {
				block_type[gr] = byte(m.b.ReadBits(2))
				switch_point[gr] = byte(m.b.ReadBits(1))
				for region := 0; region < 2; region++ {
					table_select[region][gr] = byte(m.b.ReadBits(5))
				}
				for window := 0; window < 3; window++ {
					subblock_gain[window][gr] = uint8(m.b.ReadBits(3))
				}
			} else {
				for region := 0; region < 3; region++ {
					table_select[region][gr] = byte(m.b.ReadBits(5))
				}
				region_address1[gr] = byte(m.b.ReadBits(4))
				region_address2[gr] = byte(m.b.ReadBits(3))
			}
			preflag[gr] = byte(m.b.ReadBits(1))
			scalefac_scale[gr] = byte(m.b.ReadBits(1))
			count1table_select[gr] = byte(m.b.ReadBits(1))

		}
		// The main_data follows. It does not follow the above side information in the bitstream. The main_data ends at a location in the main_data bitstream preceding the frame header of the following frame at an offset given by the value of main_data_end (see definition of main_data_end and 3-Annex Fig.3-A.7.1)
		for gr := 0; gr < 2; gr++ {
			if blocksplit_flag[gr] == 1 && block_type[gr] == 2 {
				scalefac = make([][2]uint8, switch_point_l(switch_point[gr]))
				scalefacw = make([][3][2]uint8, cblimit_short-switch_point_s(switch_point[gr]))
				for cb := 0; cb < switch_point_l(switch_point[gr]); cb++ {
					if (scfsi[cb] == 0) || (gr == 0) {
						slen := scalefactors_len(scalefac_compress[gr], block_type[gr], switch_point[gr], cb)
						scalefac[cb][gr] = uint8(m.b.ReadBits(slen))
					}
				}
				for cb := switch_point_s(switch_point[gr]); cb < cblimit_short; cb++ {
					slen := scalefactors_len(scalefac_compress[gr], block_type[gr], switch_point[gr], cb)
					for window := 0; window < 3; window++ {
						if (scfsi[cb] == 0) || (gr == 0) {
							scalefacw[cb][window][gr] = uint8(m.b.ReadBits(slen))
						}
					}
				}
			} else {
				scalefac = make([][2]uint8, cblimit)
				for cb := 0; cb < cblimit; cb++ {
					if (scfsi[cb] == 0) || (gr == 0) {
						slen := scalefactors_len(scalefac_compress[gr], block_type[gr], switch_point[gr], cb)
						scalefac[cb][gr] = uint8(m.b.ReadBits(slen))
					}
				}
			}
			Huffmancodebits := uint(part2_3_length[gr]) - part2_length(switch_point[gr], scalefac_compress[gr], block_type[gr])
			println("HUF", Huffmancodebits)
			println("MDE", main_data_end)
			/*
				for position != main_data_end {
					m.b.ReadBits(1) // ancillary_bit
				}
			//*/
		}
	}
	/* else if (mode == ModeStereo) || (mode == ModeDual) || (mode == ModeJoint) {
		main_data_end := uint16(m.b.ReadBits(9))
		private_bits := byte(m.b.ReadBits(3))
		for ch := 0; ch < 2; ch++ {
			for scfsi_band = 0; scfsi_band < 4; scfsi_band++ {
				scfsi[scfsi_band][ch] = byte(m.b.ReadBits(1))
			}
		}
		for gr := 0; gr < 2; gr++ {
			for ch := 0; ch < 2; ch++ {
				part2_3_length[gr][ch] = uint16(m.b.ReadBits(12))
				big_values[gr][ch] = uint16(m.b.ReadBits(9))
				global_gain[gr][ch] = uint16(m.b.ReadBits(8))
				scalefac_compress[gr][ch] = byte(m.b.ReadBits(4))
				blocksplit_flag[gr][ch] = byte(m.b.ReadBits(1))
				if blocksplit_flag[gr][ch] {
					block_type[gr][ch] = byte(m.b.ReadBits(2))
					switch_point[gr][ch] = uint16(m.b.ReadBits(1))
					for region := 0; region < 2; region++ {
						table_select[region][gr][ch] = byte(m.b.ReadBits(5))
					}
					for window := 0; window < 3; window++ {
						subblock_gain[window][gr][ch] = uint8(m.b.ReadBits(3))
					}
				} else {
					for region := 0; region < 3; region++ {
						table_select[region][gr][ch] = byte(m.b.ReadBits(5))
					}
					region_address1[gr][ch] = byte(m.b.ReadBits(4))
					region_address2[gr][ch] = byte(m.b.ReadBits(3))
				}
				preflag[gr][ch] = byte(m.b.ReadBits(1))
				scalefac_scale[gr][ch] = byte(m.b.ReadBits(1))
				count1table_select[gr][ch] = byte(m.b.ReadBits(1))
				// The main_data follows. It does not follow the above side information in the bitstream. The main_data endsat a location in the main_data bitstream preceding the frame header of the following frame at an offset given by thevalue of main_data_end.
			}
		}
		for gr := 0; gr < 2; gr++ {
			for ch := 0; ch < 2; ch++ {
				if blocksplit_flag[gr][ch] == 1 && block_type[gr][ch] == 2 {
					for cb := 0; cb < switch_point_l[gr][ch]; cb++ {
						if (scfsi[cb] == 0) || (gr == 0) {
							// scalefac[cb][gr][ch]0..4 bits uimsbf
						}
					}
					for cb := switch_point_s[gr][ch]; cb < cblimit_short; cb++ {
						for window := 0; window < 3; window++ {
							if (scfsi[cb] == 0) || (gr == 0) {
								// scalefac[cb][window][gr][ch] 0..4 bits uimsbf
							}
						}
					}
				} else {
					for cb := 0; cb < cblimit; cb++ {
						if (scfsi[cb] == 0) || (gr == 0) {
							// scalefac[cb][gr][ch]0..4 bits uimsbf
						}
					}
				}
				// Huffmancodebits (part2_3_length-part2_length) bits bslbf
				for position != main_data_end {
					ancillary_bit := byte(m.b.ReadBits(1))
				}
			}
		}
	}
	//*/
}

// Length returns the frame length in bytes.
func (m *MP3) Length() int {
	padding := 0
	if m.padding_bit != 0 {
		padding = 1
	}
	switch m.layer {
	case LayerI:
		return (12*m.BitrateIndex()*1000/m.SamplingIndex() + padding) * 4
	case LayerII, LayerIII:
		return 144*m.BitrateIndex()*1000/m.SamplingIndex() + padding
	default:
		return 0
	}
}

func (m *MP3) BitrateIndex() int {
	switch {
	case m.layer == LayerIII:
		switch m.bitrate_index {
		case 1:
			return 32
		case 2:
			return 40
		case 3:
			return 48
		case 4:
			return 56
		case 5:
			return 64
		case 6:
			return 80
		case 7:
			return 96
		case 8:
			return 112
		case 9:
			return 128
		case 10:
			return 160
		case 11:
			return 192
		case 12:
			return 224
		case 13:
			return 256
		case 14:
			return 320
		}
	}
	return 0
}

func (m *MP3) SamplingIndex() int {
	switch m.sampling_frequency {
	case 0:
		return 44100
	case 1:
		return 48000
	case 2:
		return 32000
	}
	return 0
}

type Layer byte

const (
	LayerI   Layer = 3
	LayerII        = 2
	LayerIII       = 1
)

func (l Layer) String() string {
	switch l {
	case LayerI:
		return "layer I"
	case LayerII:
		return "layer II"
	case LayerIII:
		return "layer III"
	default:
		return "unknown"
	}
}

type Mode byte

const (
	ModeStereo Mode = 0
	ModeJoint       = 1
	ModeDual        = 2
	ModeSingle      = 3
)

type Emphasis byte

const (
	EmphasisNone  Emphasis = 0
	Emphasis50_15          = 1
	EmphasisCCIT           = 3
)

const (
	cblimit       = 21
	cblimit_short = 12
)

func switch_point_l(b byte) int {
	if b == 0 {
		return 0
	}
	return 8
}

func switch_point_s(b byte) int {
	if b == 0 {
		return 0
	}
	return 3
}

func part2_length(switch_point, scalefac_compress, block_type byte) uint {
	slen1, slen2 := slen12(scalefac_compress)
	switch switch_point {
	case 0:
		switch block_type {
		case 0, 1, 3:
			return 11*slen1 + 10*slen2
		case 2:
			return 18*slen1 + 18*slen2
		}
	case 1:
		switch block_type {
		case 0, 1, 3:
			return 11*slen1 + 10*slen2
		case 2:
			return 17*slen1 + 18*slen2
		}
	}
	panic("unreachable")
}

func slen12(scalefac_compress byte) (slen1, slen2 uint) {
	switch scalefac_compress {
	case 0:
		return 0, 0
	case 1:
		return 0, 1
	case 2:
		return 0, 2
	case 3:
		return 0, 3
	case 4:
		return 3, 0
	case 5:
		return 1, 1
	case 6:
		return 1, 2
	case 7:
		return 1, 3
	case 8:
		return 2, 1
	case 9:
		return 2, 2
	case 10:
		return 2, 3
	case 11:
		return 3, 1
	case 12:
		return 3, 2
	case 13:
		return 3, 3
	case 14:
		return 4, 2
	case 15:
		return 4, 3
	}
	panic("unreachable")
}

func scalefactors_len(scalefac_compress, block_type, switch_point byte, cb int) uint {
	slen1, slen2 := slen12(scalefac_compress)
	switch block_type {
	case 0, 1, 3:
		if cb <= 10 {
			return slen1
		}
		return slen2
	case 2:
		switch {
		case switch_point == 0 && cb <= 5:
			return slen1
		case switch_point == 0 && cb > 5:
			return slen2
		case switch_point == 1 && cb <= 5:
			// FIX: see spec note about long windows
			return slen1
		case switch_point == 1 && cb > 5:
			return slen2
		}
	}
	panic("unreachable")
}
