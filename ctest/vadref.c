// Minimal C test: reads WAV, runs fvad_process per-frame, outputs results.
// Compile: cd ctest && gcc -std=c11 -I../../libfvad-1.0/include -o vadref vadref.c \
//   ../../libfvad-1.0/src/fvad.c ../../libfvad-1.0/src/vad/vad_core.c \
//   ../../libfvad-1.0/src/vad/vad_filterbank.c ../../libfvad-1.0/src/vad/vad_gmm.c \
//   ../../libfvad-1.0/src/vad/vad_sp.c \
//   ../../libfvad-1.0/src/signal_processing/division_operations.c \
//   ../../libfvad-1.0/src/signal_processing/energy.c \
//   ../../libfvad-1.0/src/signal_processing/get_scaling_square.c \
//   ../../libfvad-1.0/src/signal_processing/resample_48khz.c \
//   ../../libfvad-1.0/src/signal_processing/resample_by_2_internal.c \
//   ../../libfvad-1.0/src/signal_processing/resample_fractional.c \
//   ../../libfvad-1.0/src/signal_processing/spl_inl.c
#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <string.h>
#include "fvad.h"

typedef struct {
    int sample_rate;
    int num_samples;
    int16_t *data;
} wav_t;

static void read_le32(const uint8_t *p, uint32_t *v) { *v = p[0]|(p[1]<<8)|(p[2]<<16)|(p[3]<<24); }
static void read_le16(const uint8_t *p, uint16_t *v) { *v = p[0]|(p[1]<<8); }

static int read_wav(const char *path, wav_t *w) {
    FILE *f = fopen(path, "rb");
    if (!f) { perror(path); return -1; }
    fseek(f, 0, SEEK_END);
    long fsize = ftell(f);
    fseek(f, 0, SEEK_SET);
    uint8_t *buf = malloc(fsize);
    if (!buf || fread(buf, 1, fsize, f) != (size_t)fsize) { free(buf); fclose(f); return -1; }
    fclose(f);

    if (fsize < 44 || memcmp(buf, "RIFF", 4) || memcmp(buf+8, "WAVE", 4)) {
        free(buf); return -1;
    }
    memset(w, 0, sizeof(*w));
    long pos = 12;
    while (pos + 8 <= fsize) {
        char ckid[5] = {0}; memcpy(ckid, buf+pos, 4);
        uint32_t cksz; read_le32(buf+pos+4, &cksz);
        if (pos + 8 + cksz > (uint32_t)fsize) break;
        if (strcmp(ckid, "fmt ") == 0) {
            uint16_t afmt; read_le16(buf+pos+8, &afmt);
            if (afmt != 1) { free(buf); return -1; }
            uint16_t ch; read_le16(buf+pos+10, &ch);
            if (ch != 1) { free(buf); return -1; }
            read_le32(buf+pos+12, (uint32_t*)&w->sample_rate);
        } else if (strcmp(ckid, "data") == 0) {
            w->num_samples = cksz / 2;
            w->data = malloc(cksz);
            if (!w->data) { free(buf); return -1; }
            memcpy(w->data, buf+pos+8, cksz);
        }
        pos += 8 + cksz;
    }
    free(buf);
    if (!w->data) return -1;
    return 0;
}

int main(int argc, char *argv[]) {
    if (argc < 5) {
        fprintf(stderr, "Usage: %s <wav> <mode> <frame_ms> <outfile>\n", argv[0]);
        return 1;
    }
    wav_t w = {0};
    if (read_wav(argv[1], &w) != 0) return 1;
    int mode = atoi(argv[2]);
    int frame_ms = atoi(argv[3]);
    int sr = w.sample_rate;

    Fvad *vad = fvad_new();
    fvad_set_mode(vad, mode);
    fvad_set_sample_rate(vad, sr);

    int frame_size = sr * frame_ms / 1000;
    int spms = sr / 1000;
    if (frame_size != 10*spms && frame_size != 20*spms && frame_size != 30*spms) {
        fprintf(stderr, "Invalid frame size %d\n", frame_size);
        fvad_free(vad); free(w.data); return 1;
    }

    FILE *out = fopen(argv[4], "w");
    if (!out) { perror(argv[4]); fvad_free(vad); free(w.data); return 1; }

    int voice = 0, total = 0;
    for (int off = 0; off + frame_size <= w.num_samples; off += frame_size) {
        int r = fvad_process(vad, w.data + off, frame_size);
        if (r < 0) { fprintf(stderr, "err at %d\n", off); break; }
        fprintf(out, "%d\n", r);
        if (r) voice++;
        total++;
    }
    fprintf(out, "# total=%d voice=%d\n", total, voice);
    fclose(out);
    fvad_free(vad);
    free(w.data);
    return 0;
}
