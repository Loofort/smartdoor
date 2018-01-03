#include <iostream>
#include <sys/types.h>
#include <dirent.h>

#include "opencv2/highgui.hpp"
#include <dlib/opencv.h>
#include <dlib/image_processing/frontal_face_detector.h>
#include <dlib/image_processing.h>
#include <dlib/image_io.h>
#include <dlib/dnn.h>
#include <dlib/gui_widgets.h> // for debug
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

/************************ Init face database ************************/
shape_predictor init_sp(const char* modelpath) {
    shape_predictor shpr;
 //   deserialize("eyes/cv/cnn-models/shape_predictor_5_face_landmarks.dat") >> shpr;
    deserialize(modelpath) >> shpr;
    return shpr;
}
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

        /*
            image_window win;
            win.set_image(img); 
            win.clear_overlay(); 
            win.add_overlay( shape.get_rect());
            cin.get();
        */

        matrix<rgb_pixel> face_chip;
        extract_image_chip(img, get_face_chip_details(shape,150,0.25), face_chip);
        faces.push_back(move(face_chip));
    }

    if (faces.size() == 0) {
        return;
    }

    std::vector<matrix<float,0,1>> face_descriptors = net(faces);
    descriptors.push_back(face_descriptors[0]);
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

ResultArr InitPersons(const char* folder, const char* modelpath ) {
    ResultArr result={0};
    try {
        sp = init_sp(modelpath);
        result = initImages(folder);
    } catch (exception& e) {
        result.Err = strdup(e.what());
    }
    return result;
}


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
            /*
            image_window win;
            win.set_image(bestframe); 
            win.clear_overlay(); 
            win.add_overlay( bestshape.get_rect());
            */

            matrix<rgb_pixel> face_chip;
            extract_image_chip(bestframe, get_face_chip_details(bestshape,150,0.25), face_chip);
            std::vector<matrix<rgb_pixel>> faces;
            faces.push_back(face_chip);
            std::vector<matrix<float,0,1>> curr_descriptors = net(faces);

            // find similar face
            for (size_t i = 0; i < descriptors.size(); ++i) {
                cout << length(descriptors[i]-curr_descriptors[0]) << " " << names[i] << "\n";

                if (length(descriptors[i]-curr_descriptors[0]) < 0.6) {
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

