zeropad_str <- function(str){
  #useful for consistent well names
  num_str <- gsub("[A-Z,a-z]","",str)
  num_fix<-sprintf("%02d",as.numeric(num_str))
  gsub(num_str,num_fix,str)
}
