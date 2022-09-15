locate_details <- function(D){
  #fee top level directory of lot
  list.files(path=D,
             pattern="details.xml$",
             recursive=T,
             full.names=T
  )
}
