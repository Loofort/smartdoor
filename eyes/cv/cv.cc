#include <iostream>
#include <sys/types.h>
#include <dirent.h>

#include "opencv2/objdetect.hpp" // for haar cascade face detector
#include "opencv2/imgproc.hpp" // for haar cascade face detector
#include "opencv2/highgui.hpp"
#include <dlib/opencv.h>
#include <dlib/image_processing/frontal_face_detector.h>
#include <dlib/image_processing.h>
#include <dlib/image_io.h>
#include <dlib/dnn.h>
#include <dlib/image_processing/render_face_detections.h> // for debug
#include <dlib/gui_widgets.h> // for debug
#include "cv.h"
#include "stdint.h"

using namespace std;
using namespace dlib;


/********************************* CADRE *************************************/

Result NewCapture(const char *path) {
    Result result={0};

    cv::VideoCapture* capture = new cv::VideoCapture();
    if(!capture->open( path )) {
        result.Err = strdup("Could not read input video file");
    } else {
        result.Res = capture;
    }

    return result;
}
void DestroyCapture(void* res) {
    if (res) delete (cv::VideoCapture*)res;
}

Result NewCadre(void* res) {
    Result result={0};
    cv::VideoCapture* capture = (cv::VideoCapture*)(res);

    cv::Mat* frame = new cv::Mat();
    if (!capture->read(*frame))
        result.Err = strdup("get empty frame");
    else {
        // we could prepare frame - resize, gray etc.
        cv::Mat img;
        cv::cvtColor( *frame, img, cv::COLOR_BGR2GRAY );
        //resize( gray, smallImg, Size(), fx, fx, INTER_LINEAR );
        cv::equalizeHist( img, img );
        cv::cvtColor( img, *frame, cv::COLOR_GRAY2BGR );

        result.Res = frame;
    }

    return result;
}
void DestroyCadre(void* res) {
    if (res) delete (cv::Mat*)res;
}

/********************************* DETECT *************************************/
// ----------------------------------------------------------------------------------------

template <long num_filters, typename SUBNET> using con5d = con<num_filters,5,5,2,2,SUBNET>;
template <long num_filters, typename SUBNET> using con5  = con<num_filters,5,5,1,1,SUBNET>;

template <typename SUBNET> using downsampler  = relu<affine<con5d<32, relu<affine<con5d<32, relu<affine<con5d<16,SUBNET>>>>>>>>>;
template <typename SUBNET> using rcon5  = relu<affine<con5<45,SUBNET>>>;

using detect_net_type = loss_mmod<con<1,9,9,1,1,rcon5<rcon5<rcon5<downsampler<input_rgb_image_pyramid<pyramid_down<6>>>>>>>>;

// ----------------------------------------------------------------------------------------
frontal_face_detector detector = get_frontal_face_detector();
cv::CascadeClassifier haarcascade;
detect_net_type detect_net;

void InitDetectors(const char* modelpath, const char* haarpath) {
    deserialize(modelpath) >> detect_net;

    if( !haarcascade.load( haarpath )) {
        cerr << "ERROR: Could not load classifier cascade" << endl;
    }
}

ResultArr HAARDetect(void* pcadre) {  
    int minsize = 30;
    int neighbors = 6;

    ResultArr result={0};
    try {
        cv::Mat img = *(cv::Mat*)pcadre;
        /*
        cv::Mat frame = *(cv::Mat*)pcadre;
        cv::Mat img;
        cv::cvtColor( frame, img, cv::COLOR_BGR2GRAY );
        cv::equalizeHist( img, img );
        */

        std::vector<cv::Rect> dets;
        haarcascade.detectMultiScale( img, dets,
            1.1, neighbors, 0
            //|CASCADE_FIND_BIGGEST_OBJECT
            //|CASCADE_DO_ROUGH_SEARCH
            |cv::CASCADE_SCALE_IMAGE,
            cv::Size(minsize, minsize) );

        // copy to go suitable array
        Rectangle* arr = 0;
        if (dets.size() > 0 ) {
            arr = (Rectangle*)malloc(dets.size() * sizeof(Rectangle)); // use malloc since going to use free
        }
        
        for(size_t i = 0; i != dets.size(); i++) {
            cv::Rect d = dets[i];
            rectangle* r = new rectangle(d.x, d.y, d.x + d.width, d.y + d.height);
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

ResultArr NetDetect(void* pcadre) {  
    ResultArr result={0};
    try {
        cv_image<bgr_pixel> frame(*(cv::Mat*)pcadre);
        matrix<rgb_pixel> img;
        assign_image(img, frame);

        //while(img.size() < 1800*1800)
        //    pyramid_up(img);

        // todo:
        // move it to cadre intialization
        auto dets = detect_net(img);

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

ResultArr Detect(void* pcadre) {  
    ResultArr result={0};
    try {
        cv_image<bgr_pixel> frame(*(cv::Mat*)pcadre);
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

/********************************* TRACKER *************************************/

Result NewTracker(void* pcadre, void* prect) {
    Result result={0};
    try {
        cv_image<bgr_pixel> frame(*(cv::Mat*)pcadre);
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
        cv_image<bgr_pixel> frame(*(cv::Mat*)pcadre);
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

/************************ Init face database ************************/
shape_predictor sp;
anet_type net;

std::vector<matrix<float,0,1>> descriptors;
std::vector<char*> names;

void addImage(char* file) {
    array2d<rgb_pixel> img;
    load_image(img, file);

    std::vector<matrix<rgb_pixel>> faces;
    for (auto face : detector(img)) {
        auto shape = sp(img, face);

        matrix<rgb_pixel> face_chip;
        extract_image_chip(img, get_face_chip_details(shape,150,0.25), face_chip);
        faces.push_back(move(face_chip));
    }

    if (faces.size() == 0) {
        return;
    }

    std::vector<matrix<float,0,1>> face_descriptors = net(faces);
    descriptors.push_back(face_descriptors[0]);

    cout << names.size() << ": added " << file << "\n";
    names.push_back(file);
    
}

ResultArr initImages(const char* folder) {
    ResultArr result={0};

    unsigned char isFile =0x8;
    struct dirent *ent;
    DIR* dir = opendir(folder);
    if (dir == NULL) {
        result.Err = strdup("Could not read person folder");
        return result;
    }

    while ((ent = readdir (dir)) != NULL) {
        if ( ent->d_type == isFile) {
            char* file = (char*)malloc(strlen(folder)+ strlen(ent->d_name)+1);
            if (file == NULL) {
                result.Err = strdup("can't create person path: malloc failed");
                return result; 
            }

            strcpy(file, folder);
            strcat(file, ent->d_name);
            addImage(file);

            //todo: free ent ?
        }
    }

    char** cnames = (char**)malloc(names.size() * sizeof(char*));
    for (size_t i = 0; i < names.size(); ++i) {
        cnames[i] = names[i];
    }

    result.Cnt = names.size();
    result.Res = cnames;
    return result;
}

ResultArr InitPersons(const char* folder, const char* modelpath, const char* netpath) {
    ResultArr result={0};
    try {
        deserialize(modelpath) >> sp;
        deserialize(netpath) >> net;
        result = initImages(folder);
    } catch (exception& e) {
        result.Err = strdup(e.what());
    }
    return result;
}


Result GetShape(void* pcadre, void* prect) {
    Result result={0};
    try {
        cv_image<bgr_pixel> frame(*(cv::Mat*)pcadre);
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

double kk=10.0;
Result Recognize (void* pcadre[], void* prect[], int len ) {
    Result result={0};
    try {
        cv_image<bgr_pixel> bestframe;
        full_object_detection bestshape;
        bool ok;
        
        // chose the best face
        // currently just select the biggest one
        for (int i = 0; i < len; i++) {
            cv_image<bgr_pixel> frame = (*(cv::Mat*)pcadre[i]);
            rectangle rect = *(rectangle*)(prect[i]);
            auto shape = sp(frame, rect);
            if (shape.num_parts() < 5) {
                continue;
            }
            
            ok = true;
            if (shape.get_rect().area() > bestshape.get_rect().area()) {
                bestshape = shape;
                bestframe = frame;
            }
        }

        if (ok) {
            
            /*
            image_window win;
            win.set_image(bestframe); 
            win.clear_overlay(); 
            win.add_overlay(render_face_detections(bestshape));
            //cin.get();
            */

            
            /*
            array2d<rgb_pixel> img;
            extract_image_chip(bestframe, bestshape.get_rect(), img);
            array2d<matrix<float,31,1> > hog;
            extract_fhog_features(img, hog);
            //matrix<unsigned char> df = draw_fhog(hog);
            image_window hogwin(draw_fhog(hog), "Learned fHOG detector");
            */


            

            matrix<rgb_pixel> face_chip;
            extract_image_chip(bestframe, get_face_chip_details(bestshape,150,0.25), face_chip);
            
            image_window win;
            win.set_image(face_chip); 
            //cin.get();

            std::vector<matrix<rgb_pixel>> faces;
            faces.push_back(face_chip);
            std::vector<matrix<float,0,1>> curr_descriptors = net(faces);

            

            // find similar face
            for (size_t i = 0; i < descriptors.size(); ++i) {
                double k = length(descriptors[i]-curr_descriptors[0]);
                if ( k < 0.6) {
                    if (k < kk || kk == 0) {
                        kk = k;
                    }
                    cout << i << ":" << k << "(" << kk << ") " << bestshape.num_parts() << " \n";
                    result.Res = names[i];
                    return result;
                }
            }

            //cin.get();
        }
    } catch (exception& e) {
        result.Err = strdup(e.what());
    }
    return result;
}

/********************************* helpers for debug **************************************/


void SaveRFrame(void* pcadre, void* prect, const char *path) {
    cv_image<bgr_pixel> frame(*(cv::Mat*)pcadre);
    rectangle rect = *(rectangle*)(prect);

    draw_rectangle(frame, rect, dlib::rgb_pixel(255, 0, 0), 1);
        
    save_png(frame, path);
}

void ShowRFrame(void* pcadre, void* prect) {
    cv_image<bgr_pixel> frame(*(cv::Mat*)pcadre);
    rectangle rect = *(rectangle*)(prect);
    
    image_window win;
    win.set_image(frame); 
    win.clear_overlay(); 
    win.add_overlay(rect);
    cin.get();
}