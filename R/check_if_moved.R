check_if_moved<-function(Lot){
# see if data has been moved to G-drive
type <- substr(Lot,1,1)
sub <- c("W"="XFe96","C"="XFp","B"="XFe24")[type]
path <- file.path("G:","Spotting","Logging",sub)
lotdirs <- list.dirs(path=path,recursive = F)
any(grepl(Lot,lotdirs))
}
