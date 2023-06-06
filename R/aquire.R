aquire<- function(u){
  require(parallel)
  fls <- locate_details(u)
  n_cores<-detectCores()
  cl <- makeCluster(n_cores)
  df <- parLapply(cl, fls, function(u){
    try(vis::parse(u))
  }) %>% 
    dplyr::bind_rows()
  parallel::stopCluster(cl)
  df
}
