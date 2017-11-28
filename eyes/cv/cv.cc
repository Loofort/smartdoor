#include <iostream>
#include "opencv2/highgui.hpp"
#include <dlib/image_processing/frontal_face_detector.h>
#include <dlib/opencv.h>
#include <dlib/image_processing.h>
#include <dlib/dnn.h>
#include "cv.h"
#include "stdint.h"

using namespace std;
using namespace cv;
using namespace dlib;


/********************************* CADRE *************************************/

Result NewCapture(const char *path) {
    Result result={0};

    VideoCapture* capture = new VideoCapture();
    if(!capture->open( path )) {
        result.Err = strdup("Could not read input video file");
    } else {
        result.Res = capture;
    }

    return result;
}
void DestroyCapture(void* res) {
    if (res) delete (VideoCapture*)res;
}

Result NewCadre(void* res) {
    Result result={0};
    VideoCapture* capture = (VideoCapture*)(res);

    Mat* frame = new Mat();
    if (!capture->read(*frame))
        result.Err = strdup("get empty frame");
    else {
        // we could prepare frame - resize, gray etc.
        /*Mat gray, smallImg;
        cvtColor( img, gray, COLOR_BGR2GRAY );
        resize( gray, smallImg, Size(), fx, fx, INTER_LINEAR );
        equalizeHist( smallImg, smallImg );
        */

        result.Res = frame;
    }

    return result;
}
void DestroyCadre(void* res) {
    if (res) delete (Mat*)res;
}

/********************************* TRACKER *************************************/

frontal_face_detector detector = get_frontal_face_detector();

ResultArr Detect(void* pcadre) {  
    ResultArr result={0};
    try {
        cv_image<bgr_pixel> frame(*(Mat*)pcadre);
        //pyramid_up(frame);

        // todo:
        // move it to cadre intialization
        std::vector<rectangle> dets = detector(frame);

        // copy to go suitable array
        Rectangle* arr = 0;
        if (dets.size() > 0 ) {
            arr = (Rectangle*)malloc(dets.size() * sizeof(Rectangle)); // use malloc since going to use free
        }
        for(std::vector<rectangle>::size_type i = 0; i != dets.size(); i++) {
            rectangle* r = new rectangle();
            *r = dets[i];
            
            Rectangle rect = Rectangle {
                .Rect = r,
                .Left = r->left(),
                .Top = r->top(),
                .Right = r->right(),
                .Bottom = r->bottom()
            };

            arr[i]=rect;
        }

        result.Cnt = dets.size();
        result.Res = arr;

    } catch (exception& e) {
        result.Err = strdup(e.what());
    }
    return result;
}

void DestroyRect(void* res) {
    if (res) delete (rectangle*)res;
}

Result NewTracker(void* pcadre, void* prect) {
    Result result={0};
    try {
        cv_image<bgr_pixel> frame(*(Mat*)pcadre);
        rectangle rect = *(rectangle*)(prect);

        correlation_tracker* tracker = new correlation_tracker();
        tracker->start_track(frame, rect);

        result.Res = tracker;
    } catch (exception& e) {
        result.Err = strdup(e.what());
    }
    return result;
}

void DestroyTracker(void* res) {
    if (res) delete (correlation_tracker*)res;
}

Result UpdateTracker(void* ptracker, void* pcadre) {
    Result result={0};
    try {
        cv_image<bgr_pixel> frame(*(Mat*)pcadre);
        ((correlation_tracker*)ptracker)->update(frame);

        rectangle rect = ((correlation_tracker*)ptracker)->get_position();
        rectangle* r = new rectangle();
        *r = rect;

        //create go rect
        Rectangle* gorect = (Rectangle*)malloc(sizeof(Rectangle));
        *gorect = Rectangle {
            .Rect = r,
            .Left = r->left(),
            .Top = r->top(),
            .Right = r->right(),
            .Bottom = r->bottom()
        };

        result.Res = gorect;
    } catch (exception& e) {
        result.Err = strdup(e.what());
    }
    return result;
}

/************************************ Person *******************************************/

// ------------------------------- Define face identification cnn ---------------------------------------------------------
template <template <int,template<typename>class,int,typename> class block, int N, template<typename>class BN, typename SUBNET>
using residual = add_prev1<block<N,BN,1,tag1<SUBNET>>>;

template <template <int,template<typename>class,int,typename> class block, int N, template<typename>class BN, typename SUBNET>
using residual_down = add_prev2<avg_pool<2,2,2,2,skip1<tag2<block<N,BN,2,tag1<SUBNET>>>>>>;

template <int N, template <typename> class BN, int stride, typename SUBNET> 
using block  = BN<con<N,3,3,1,1,relu<BN<con<N,3,3,stride,stride,SUBNET>>>>>;

template <int N, typename SUBNET> using ares      = relu<residual<block,N,affine,SUBNET>>;
template <int N, typename SUBNET> using ares_down = relu<residual_down<block,N,affine,SUBNET>>;

template <typename SUBNET> using alevel0 = ares_down<256,SUBNET>;
template <typename SUBNET> using alevel1 = ares<256,ares<256,ares_down<256,SUBNET>>>;
template <typename SUBNET> using alevel2 = ares<128,ares<128,ares_down<128,SUBNET>>>;
template <typename SUBNET> using alevel3 = ares<64,ares<64,ares<64,ares_down<64,SUBNET>>>>;
template <typename SUBNET> using alevel4 = ares<32,ares<32,ares<32,SUBNET>>>;

using anet_type = loss_metric<fc_no_bias<128,avg_pool_everything<
                            alevel0<
                            alevel1<
                            alevel2<
                            alevel3<
                            alevel4<
                            max_pool<3,3,2,2,relu<affine<con<32,7,7,2,2,
                            input_rgb_image_sized<150>
                            >>>>>>>>>>>>;

// ----------------------------------------------------------------------------------------

shape_predictor init_sp() {
    shape_predictor shpr;
    deserialize("eyes/cv/cnn-models/shape_predictor_5_face_landmarks.dat") >> shpr;
    return shpr;
}
shape_predictor sp=init_sp();
anet_type net;

Result GetShape(void* pcadre, void* prect) {
    Result result={0};
    try {
        cv_image<bgr_pixel> frame(*(Mat*)pcadre);
        rectangle rect = *(rectangle*)(prect);

        auto shape = sp(frame, rect);
	    matrix<rgb_pixel>* face_chip;
        extract_image_chip(frame, get_face_chip_details(shape,150,0.25), *face_chip);
        
	    result.Res = face_chip;
    } catch (exception& e) {
        result.Err = strdup(e.what());
    }
    return result;
}

Result Recognize (void* pcadre[], void* prect[], int len ) {
    Result result={0};
    try {
        cv_image<bgr_pixel> bestframe;
        full_object_detection bestshape;
        bool ok;
        
        // chose the best face
        // currently just select the biggest one
        for (int i = 0; i < len; i++) {
            cv_image<bgr_pixel> frame = (*(Mat*)pcadre[i]);
            rectangle rect = *(rectangle*)(prect[i]);
            auto shape = sp(frame, rect);
            
            if (shape.num_parts() < 5) 
                continue;
            
            ok = true;
            if (shape.get_rect().area() > bestshape.get_rect().area()) {
                bestshape = shape;
                bestframe = frame;
            }
        }

        if (ok) {
            matrix<rgb_pixel>* face_chip;
            extract_image_chip(bestframe, get_face_chip_details(bestshape,150,0.25), *face_chip);
            
            std::vector<matrix<rgb_pixel>> faces;
            auto f = *(face_chip);
            faces.push_back(f);
            std::vector<matrix<float,0,1>> face_descriptors = net(faces);

            // find similar face 


        }
    } catch (exception& e) {
        result.Err = strdup(e.what());
    }
    return result;
}