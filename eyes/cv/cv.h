#include "stdint.h"

#ifdef __cplusplus
extern "C" {
#endif

typedef struct Result {
    const char *Err;
    void *Res;
} Result;

typedef struct ResultArr {
    const char *Err;
    void *Res;
    uint32_t Cnt;
} ResultArr;

typedef struct Rectangle {
    void *Rect;
    long Left;
    long Top;
    long Right;
    long Bottom;
} Rectangle;


Result NewCapture(const char *path);
void DestroyCapture(void* res);

Result NewCadre(void* res);
void DestroyCadre(void* res);

ResultArr Detect(void* pcadre);
void DestroyRect(void* res);

Result NewTracker(void* pcadre, void* prect);
Result UpdateTracker(void* ptracker, void* pcadre);
void DestroyTracker(void* res);

Result Recognize (void* pcadre[], void* prect[], int len );
ResultArr InitPersons(const char* folder, const char* modelpath);

#ifdef __cplusplus
}
#endif