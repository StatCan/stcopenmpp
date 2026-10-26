#pragma once
#include "case-based.h"

bool using_fictive_person();
region_t region_of_province(int province);

int pps_sampling(double u, double p[], int number_of_categories);
double cubic_spline(double age_or_time, double *knots_location, double *cubic_spline_coef, double *cubic_spline_companion, int number_of_splines);
double cubic_spline(double age_or_time, double *knots_location, double *cubic_spline_coef, double *cubic_spline_companion);

double update_mport_v2_spline_age(int x, sex_t sexe);
double update_mport_v2_spline_bmi(double x, sex_t sexe);

alcohol5_t get_alcohol5_from_start_pop(int alcohol_on_start_pop, sex_t sexe);
