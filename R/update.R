update <- function() {
  library(vis)
  library(dplyr)
  top <- "/mnt/LSAG/Spotting/Logging"
  csvdir <- file.path(top, "CSVs")
  csvs <- list.files(csvdir, full.names = T, pattern = '.[.]csv')
  done <- gsub("_MV[.]csv", "", basename(csvs))
  
  drs <- file.path(top, c("XFe24", "XFp", "XFe96"))
  subs <- unlist(lapply(drs, list.dirs, recursive = F))
  
  zzz <- lapply(subs, function(u) {
    lot <- basename(u)
    msg <- "Lot %s% is done."
    fin <- gsub("%s%", lot, msg)
    if (lot %in% done) {
      message(fin)
    } else{
      x <- vis::aquire(u)
      dirx <- file.path(top, "CSVs/%s%_MV.csv")
      write.csv(x, gsub("%s%", lot, dirx), row.names = F)
      message(fin)
    }
  })
}
