parse_lot_context <- function(DIR) {
  require(parallel)
  fls <- fs::dir_ls(path = DIR,
                    regexp =  "context.xml$",
                    recurse = T)
  n_cores <- detectCores()
  cl <- makeCluster(n_cores)
  df <- dplyr::bind_rows(parLapply(cl, fls, context))
  parallel::stopCluster(cl)
  df
}
