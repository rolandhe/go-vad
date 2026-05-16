// Tests downBy2ShortToInt specifically.
#include <stdio.h>
#include <stdint.h>
#include "../libfvad/src/signal_processing/resample_by_2_internal.h"

int main() {
    int16_t in[480];
    // Fill with same "speech" pattern used in test.
    for (int i=0;i<480;i++) in[i] = (int16_t)(i*i);

    int32_t out[240];
    int32_t state[8] = {0};

    WebRtcSpl_DownBy2ShortToInt(in, 480, out, state);

    printf("state after first 480:");
    for (int i=0;i<8;i++) printf(" %d", state[i]);
    printf("\nout first 8:");
    for (int i=0;i<8;i++) printf(" %d", out[i]);
    printf("\nout last 8:");
    for (int i=232;i<240;i++) printf(" %d", out[i]);
    printf("\n");

    // Second call with new data.
    int16_t in2[480];
    for (int i=0;i<480;i++) in2[i] = (int16_t)((480+i)*(480+i));

    WebRtcSpl_DownBy2ShortToInt(in2, 480, out, state);

    printf("state after second 480:");
    for (int i=0;i<8;i++) printf(" %d", state[i]);
    printf("\nout first 8:");
    for (int i=0;i<8;i++) printf(" %d", out[i]);
    printf("\n");

    return 0;
}
