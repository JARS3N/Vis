mv2csv <- function() {
require(dplyr)
  # relevant directory paths
  head <- file.path("G:", "Spotting", "Logging")
  csvdir <- file.path(head, "CSVs")
  drs <-
    file.path("G:/Spotting/Logging", c("XFe24", "XFp", "XFe96"))
  # get list of csvs already processed
  csvs <- list.files(csvdir, full.names = T, pattern = '.[.]csv')
  # get lot from list of those processed
  done <- gsub("_MV[.]csv", "", basename(csvs))
  # look for mv lot directories
  subs <- unlist(lapply(drs, list.dirs, recursive = F))
  # loop through each of the subs above
  # check if they are exist in the done list
  # if not process them and save as csv to csv directory
  lapply(subs, function(u) {
    lot <- basename(u)
    msg <- "Lot %s% is done."
    fin <- gsub("%s%", lot, msg)
    if (lot %in% done) {
      message(fin)
    } else{
      x <- vis::aquire(u)
      dirx <- file.path(csvdir, "%s%_MV.csv")
      write.csv(x, gsub("%s%", lot, dirx), row.names = F)
      message(fin)
    }
  })
}
}
