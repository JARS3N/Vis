check_if_moved<-function(Lot,value=F){
# see if data has been moved to G-drive
type <- substr(Lot,1,1)
sub <- c( 
         B = "XFe24",
         C = "XFp",
         W = "XFe96",
         X = "XFe96",
         Y = "XFe24",
         Z = "XFp"
         )[type]
path <- file.path("G:","Spotting","Logging",sub)
lotdirs <- list.dirs(path=path,recursive = F)
 if(value==T){return(grep(Lot,lotdirs,value=T))}
any(grepl(Lot,lotdirs))
}
